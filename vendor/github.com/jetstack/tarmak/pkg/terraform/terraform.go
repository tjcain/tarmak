// Copyright Jetstack Ltd. See LICENSE for details.
package terraform

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"

	"github.com/hashicorp/terraform/command"
	"github.com/kardianos/osext"
	"github.com/sirupsen/logrus"

	"github.com/jetstack/tarmak/pkg/tarmak/interfaces"
	"github.com/jetstack/tarmak/pkg/tarmak/utils"
	"github.com/jetstack/tarmak/pkg/tarmak/utils/consts"
	"github.com/jetstack/tarmak/pkg/tarmak/utils/input"
	"github.com/jetstack/tarmak/pkg/terraform/plan"
	"github.com/jetstack/tarmak/pkg/terraform/providers/tarmak/rpc"
)

const (
	debugShell       = "debug-shell"
	remoteStateError = "Error locking destination state: Error acquiring the state lock: ConditionalCheckFailedException: The conditional request failed"
)

type Terraform struct {
	log    *logrus.Entry
	tarmak interfaces.Tarmak
	ctx    interfaces.CancellationContext

	socketPath string
	prepared   bool
}

func New(tarmak interfaces.Tarmak) *Terraform {
	log := tarmak.Log().WithField("module", "terraform")

	return &Terraform{
		log:    log,
		tarmak: tarmak,
		ctx:    tarmak.CancellationContext(),
	}
}

// this method perpares the terraform plugins folder. This folder contains
// terraform providers and provisioners in general. We are pointing through
// symlinks to the tarmak binary, which contains all relevant providers
func (t *Terraform) preparePlugins(c interfaces.Cluster) error {
	binaryPath, err := osext.Executable()
	if err != nil {
		return fmt.Errorf("error finding tarmak executable: %s", err)
	}

	pluginPath := t.pluginPath(c)
	if err := os.MkdirAll(pluginPath, 0755); err != nil {
		return err
	}

	for providerName, _ := range InternalProviders {
		destPath := filepath.Join(pluginPath, fmt.Sprintf("terraform-provider-%s", providerName))
		if stat, err := os.Lstat(destPath); err != nil && !os.IsNotExist(err) {
			return err
		} else if err == nil {
			if (stat.Mode() & os.ModeSymlink) == 0 {
				return fmt.Errorf("%s is not a symbolic link", destPath)
			}

			if linkPath, err := os.Readlink(destPath); err != nil {
				return err
			} else if linkPath == binaryPath {
				// link points to correct destination
				continue
			}

			err := os.Remove(destPath)
			if err != nil {
				return err
			}
		}

		err := os.Symlink(
			binaryPath,
			destPath,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// plugin path that stores terraform providers binaries
func (t *Terraform) pluginPath(c interfaces.Cluster) string {
	return filepath.Join(t.codePath(c), command.DefaultPluginVendorDir)
}

// code path to store terraform modules and files
func (t *Terraform) codePath(c interfaces.Cluster) string {
	return filepath.Join(c.ConfigPath(), "terraform")
}

func (t *Terraform) Prepare(cluster interfaces.Cluster) error {
	if err := t.checkDone(); err != nil {
		return err
	}

	// generate tf code
	if err := t.GenerateCode(cluster); err != nil {
		return fmt.Errorf("failed to generate code: %s", err)
	}

	if err := t.checkDone(); err != nil {
		return err
	}

	// symlink tarmak plugins into folder
	if err := t.preparePlugins(cluster); err != nil {
		return fmt.Errorf("failed to prepare plugins: %s", err)
	}

	if err := t.checkDone(); err != nil {
		return err
	}

	type setupCommand struct {
		log, name string
		args      []string
	}

	var furtherErrorContext string
	stderrReader, stderrWriter := io.Pipe()
	stderrScanner := bufio.NewScanner(stderrReader)
	go func() {
		for stderrScanner.Scan() {
			if strings.Contains(stderrScanner.Text(), remoteStateError) {
				furtherErrorContext = fmt.Sprintf(`%s
this error is often caused due to the remote state being destroyed and can be fixed by manually syncing both local and remote states`, furtherErrorContext)
			}
			t.log.WithField("std", "err").Debug(stderrScanner.Text())
		}
	}()

	for _, c := range []setupCommand{
		{log: "initialising terraform", name: "init", args: []string{
			"terraform",
			"init",
			"-get-plugins=false",
			"-input=false",
		}},
		{log: "validating terraform code", name: "validate", args: []string{
			"terraform",
			"validate",
		}},
	} {
		t.log.Info(c.log)

		if err := t.command(cluster, c.args, nil, nil, stderrWriter); err != nil {
			return fmt.Errorf("error running terraform %s: %s%s", c.name, err, furtherErrorContext)
		}

		if err := t.checkDone(); err != nil {
			return err
		}
	}

	return nil
}

// this method resets the socket path and the prepare step
// you need to call this method when you do multiple Terraform runs of different clusters
func (t *Terraform) ResetTerraformWrapper() {
	t.socketPath = ""
	t.prepared = false
}

func (t *Terraform) terraformWrapper(cluster interfaces.Cluster, command string, args []string) error {
	if t.socketPath == "" {
		f, err := ioutil.TempFile(os.TempDir(), "tarmak.sock")
		if err != nil {
			return fmt.Errorf("failed to create socket file: %v", err)
		}
		t.socketPath = f.Name()
	}

	if !t.prepared {
		if err := t.Prepare(cluster); err != nil {
			return fmt.Errorf("failed to prepare terraform: %s", err)
		}

		t.prepared = true
	}

	// listen to rpc
	stopRpcCh := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := rpc.ListenUnixSocket(
			rpc.New(t.tarmak.Cluster()),
			t.socketPath,
			stopRpcCh,
		); err != nil {
			t.log.Fatalf("error listening to unix socket: %s", err)
		}
	}()

	t.log.Infof("running %s", command)

	// command
	if command == debugShell {
		dir := t.codePath(cluster)
		envVars, err := t.envVars(cluster)
		if err != nil {
			return err
		}

		// use $SHELL if available, fall back to /bin/sh
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
			envVars = append(envVars, fmt.Sprintf("PS1=[%s]$ ", dir))
		}

		cmd := exec.Command(shell)
		cmd.Dir = dir
		// envVars variables will override any shell envs with equal key
		cmd.Env = append(os.Environ(), envVars...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		cmd.Run()
	} else if command != "" {
		cmdArgs := []string{
			"terraform",
			command,
		}
		cmdArgs = append(cmdArgs, args...)

		if err := t.command(
			cluster,
			cmdArgs,
			nil,
			nil,
			nil,
		); err != nil {
			return err
		}
	}

	close(stopRpcCh)
	wg.Wait()

	return nil
}

func (t *Terraform) envVars(cluster interfaces.Cluster) ([]string, error) {
	envVars := []string{
		"TF_IN_AUTOMATION=1",
	}

	// get environment variables necessary for provider
	if environmentProvider, err := cluster.Environment().Provider().Environment(); err != nil {
		return []string{}, fmt.Errorf("error getting environment secrets from provider: %s", err)
	} else {
		envVars = append(envVars, environmentProvider...)
	}

	envVars = append(envVars, fmt.Sprintf("TF_LOG=%s", os.Getenv("TF_LOG")))

	return envVars, nil
}

func (t *Terraform) command(cluster interfaces.Cluster, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	envVars, err := t.envVars(cluster)
	if err != nil {
		return err
	}

	binaryPath, err := osext.Executable()
	if err != nil {
		return fmt.Errorf("error finding tarmak executable: %s", err)
	}

	cmd := exec.Command(
		binaryPath,
		args...,
	)

	// This ensures that processes are run in different process groups so a
	// signal to the parent process is not propagated to the children. This is
	// needed to control signaling and ensure graceful shutdown of
	// subprocesses.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// forward stdout
	if stdout == nil {
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}

		stdoutScanner := bufio.NewScanner(stdoutPipe)
		go func() {
			for stdoutScanner.Scan() {
				t.log.WithField("std", "out").Debug(stdoutScanner.Text())
			}
		}()
	} else {
		cmd.Stdout = stdout
	}

	// forward stderr
	if stderr == nil {
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return err
		}

		stderrScanner := bufio.NewScanner(stderrPipe)
		go func() {
			for stderrScanner.Scan() {
				t.log.WithField("std", "err").Debug(stderrScanner.Text())
			}
		}()
	} else {
		cmd.Stderr = stderr
	}

	cmd.Stdin = stdin
	cmd.Dir = t.codePath(cluster)
	cmd.Env = envVars

	if err := cmd.Start(); err != nil {
		return err
	}

	complete := make(chan struct{})
	go func() {
		err = cmd.Wait()
		close(complete)
	}()

	select {
	case <-t.tarmak.CancellationContext().Done():
		if cmd.Process != nil {
			cmd.Process.Signal(t.tarmak.CancellationContext().Signal())
		}
		<-complete

	case <-complete:
	}

	return err
}

// this checks if an error is coming from an exec failing with exit code 2,
// this is typicall for a terraform plan that has changes
func errIsTerraformPlanChangesNeeded(err error) bool {
	if err == nil {
		return false
	} else if exitError, ok := err.(*exec.ExitError); !ok {
		return false
	} else if status, ok := exitError.ProcessState.Sys().(syscall.WaitStatus); !ok {
		return false
	} else if status.ExitStatus() != 2 {
		return false
	}

	return true
}

func (t *Terraform) Plan(cluster interfaces.Cluster, preApply bool) (changesNeeded bool, err error) {
	var tfPlan *plan.Plan
	planPath := t.tarmak.ClusterFlags().Apply.PlanFileLocation
	changesNeeded = true

	// If we are not doing an apply after this plan OR we are not using a custom
	// plan file, we need to run a terraform plan.
	customPlanFile := planPath != consts.DefaultPlanLocationPlaceholder
	if !preApply || !customPlanFile {
		planPath, err = t.planFileStore(cluster)
		if err != nil {
			return changesNeeded, err
		}

		changesNeeded, tfPlan, err = t.planWrapper(cluster, planPath)
		if err != nil {
			return changesNeeded, err
		}
	} else {
		t.log.Infof("using custom plan file %s", planPath)

		tfPlan, err = plan.New(planPath)
		if err != nil {
			return changesNeeded, fmt.Errorf("error while trying to read plan file: %s", err)
		}
	}

	if tfPlan.UpdatingPuppet() {
		t.log.Info("tainting legacy s3 puppet module object to force update")

		if err := t.terraformWrapper(cluster, "taint", []string{
			"-allow-missing", "-module=kubernetes",
			cluster.Environment().Provider().LegacyPuppetTFName(),
		}); err != nil {
			return changesNeeded, err
		}

		t.log.Info("running plan again to update plan file against new state")

		changesNeeded, tfPlan, err = t.planWrapper(cluster, planPath)
		if err != nil {
			return changesNeeded, err
		}
	}

	destroyingEBSVolume, ebsVolumesToDestroy := tfPlan.IsDestroyingEBSVolume()
	if !destroyingEBSVolume {
		return changesNeeded, nil
	}

	destroyStr := fmt.Sprintf(
		"the following EBS volumes will be destroyed during the next apply: [%s]",
		strings.Join(ebsVolumesToDestroy, ", "))

	// We exit early here since we are only doing a plan. Bubble the ebs error up.
	if !preApply {
		return changesNeeded, errors.New(destroyStr)
	}

	if t.tarmak.ClusterFlags().Apply.AutoApproveDeletingData && t.tarmak.ClusterFlags().Apply.AutoApprove {
		t.log.Warnf("auto approved deleting, %s", destroyStr)
		return changesNeeded, nil
	}

	query := fmt.Sprintf("%s\nThis cannot be undone. Are you sure you want to continue?", destroyStr)
	d, err := input.New(os.Stdin, os.Stdout).AskYesNo(&input.AskYesNo{
		Default: false,
		Query:   query,
	})
	if err != nil {
		return changesNeeded, err
	}

	if !d {
		return changesNeeded, fmt.Errorf("error: %s", destroyStr)
	}

	t.log.Warn(destroyStr)
	return changesNeeded, nil
}

func (t *Terraform) Apply(cluster interfaces.Cluster) (hasChanged bool, err error) {
	// generate a plan
	changesNeeded, err := t.Plan(cluster, true)
	if err != nil || !changesNeeded {
		return false, err
	}

	// break after sigterm
	select {
	case <-t.ctx.Done():
		return changesNeeded, t.ctx.Err()
	default:
	}

	planFilePath, err := t.planFileLocation(cluster)
	if err != nil {
		return false, err
	}

	// apply necessary at this point
	return changesNeeded, t.terraformWrapper(
		cluster,
		"apply",
		[]string{planFilePath},
	)
}

func (t *Terraform) planWrapper(cluster interfaces.Cluster, planPath string) (changesNeeded bool, tfPlan *plan.Plan, err error) {
	err = t.terraformWrapper(
		cluster,
		"plan",
		[]string{"-detailed-exitcode", "-input=false", fmt.Sprintf("-out=%s", planPath)},
	)

	changesNeeded = errIsTerraformPlanChangesNeeded(err)
	if err != nil && !changesNeeded {
		return false, nil, err
	}

	tfPlan, err = plan.New(planPath)
	if err != nil {
		return changesNeeded, nil, fmt.Errorf("error while trying to read plan file: %s", err)
	}

	return changesNeeded, tfPlan, err
}

func (t *Terraform) Destroy(cluster interfaces.Cluster) error {
	return t.terraformWrapper(
		cluster,
		"destroy",
		[]string{"-force", "-refresh=false"},
	)
}

func (t *Terraform) ForceUnlock(cluster interfaces.Cluster, lockID string) error {
	return t.terraformWrapper(
		cluster,
		"force-unlock",
		[]string{"-force", lockID},
	)
}

func (t *Terraform) Shell(cluster interfaces.Cluster) error {
	if err := t.terraformWrapper(cluster, debugShell, nil); err != nil {
		return err
	}

	return nil
}

// convert interface map to terraform.tfvars format
func MapToTerraformTfvars(input map[string]interface{}) (output string, err error) {
	var buf bytes.Buffer

	for key, value := range input {
		switch v := value.(type) {
		case map[string]string:
			_, err := buf.WriteString(fmt.Sprintf("%s = {\n", key))
			if err != nil {
				return "", err
			}

			keys := make([]string, len(v))
			pos := 0
			for key, _ := range v {
				keys[pos] = key
				pos++
			}
			sort.Strings(keys)
			for _, key := range keys {
				_, err := buf.WriteString(fmt.Sprintf("  %s = \"%s\"\n", key, v[key]))
				if err != nil {
					return "", err
				}
			}

			_, err = buf.WriteString("}\n")
			if err != nil {
				return "", err
			}
		case []string:
			values := make([]string, len(v))
			for pos, _ := range v {
				values[pos] = fmt.Sprintf(`"%s"`, v[pos])
			}
			_, err := buf.WriteString(fmt.Sprintf("%s = [%s]\n", key, strings.Join(values, ", ")))
			if err != nil {
				return "", err
			}
		case string:
			_, err := buf.WriteString(fmt.Sprintf("%s = \"%s\"\n", key, v))
			if err != nil {
				return "", err
			}
		case int:
			_, err := buf.WriteString(fmt.Sprintf("%s = %d\n", key, v))
			if err != nil {
				return "", err
			}
		case *net.IPNet:
			_, err := buf.WriteString(fmt.Sprintf("%s = \"%s\"\n", key, v.String()))
			if err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("ignoring unknown var key='%s' type='%#+v'", key, v)
		}
	}
	return buf.String(), nil
}

func (t *Terraform) checkDone() error {
	select {
	case <-t.ctx.Done():
		return t.ctx.Err()
	default:
		return nil
	}
}

func (t *Terraform) Cleanup() error {
	if t.socketPath != "" {
		return os.RemoveAll(t.socketPath)
	}

	return nil
}

// location to store the output of the plan executable file during plan
func (t *Terraform) planFileStore(cluster interfaces.Cluster) (string, error) {
	p := t.tarmak.ClusterFlags().Plan.PlanFileStore
	if p == consts.DefaultPlanLocationPlaceholder {
		return t.defaultPlanPath(cluster), nil
	}

	return utils.Expand(p)
}

// location to use as the plan executable file during apply
func (t *Terraform) planFileLocation(cluster interfaces.Cluster) (string, error) {
	p := t.tarmak.ClusterFlags().Apply.PlanFileLocation
	if p == consts.DefaultPlanLocationPlaceholder {
		return t.defaultPlanPath(cluster), nil
	}

	return utils.Expand(p)
}

func (t *Terraform) defaultPlanPath(cluster interfaces.Cluster) string {
	return filepath.Join(t.codePath(cluster), consts.TerraformPlanFile)
}
