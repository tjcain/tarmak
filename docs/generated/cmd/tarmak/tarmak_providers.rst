.. _tarmak_providers:

tarmak providers
----------------

Operations on providers

Synopsis
~~~~~~~~


Operations on providers

Options
~~~~~~~

::

  -h, --help   help for providers

Options inherited from parent commands
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

::

  -c, --config-directory string                          config directory for tarmak's configuration (default "~/.tarmak")
      --current-cluster string                           override the current cluster set in the config
      --ignore-missing-public-key-tags ssh_known_hosts   ignore missing public key tags on instances, by falling back to populating ssh_known_hosts with the first connection (default true)
      --keep-containers                                  do not clean-up terraform/packer containers after running them
      --public-api-endpoint                              Override kubeconfig to point to cluster's public API endpoint
  -v, --verbose                                          enable verbose logging
      --wing-dev-mode                                    use a bundled wing version rather than a tagged release from GitHub

SEE ALSO
~~~~~~~~

* `tarmak <tarmak.html>`_ 	 - Tarmak is a toolkit for provisioning and managing Kubernetes clusters.
* `tarmak providers init <tarmak_providers_init.html>`_ 	 - Initialize a provider
* `tarmak providers list <tarmak_providers_list.html>`_ 	 - Print a list of providers
* `tarmak providers validate <tarmak_providers_validate.html>`_ 	 - Validate provider(s) used by current cluster

