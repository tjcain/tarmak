// +build !ignore_autogenerated

// Copyright Jetstack Ltd. See LICENSE for details.

// Code generated by defaulter-gen. DO NOT EDIT.

package v1alpha1

import (
	clusterv1alpha1 "github.com/jetstack/tarmak/pkg/apis/cluster/v1alpha1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// RegisterDefaults adds defaulters functions to the given scheme.
// Public to allow building arbitrary schemes.
// All generated defaulters are covering - they call all nested defaulters.
func RegisterDefaults(scheme *runtime.Scheme) error {
	scheme.AddTypeDefaultingFunc(&Config{}, func(obj interface{}) { SetObjectDefaults_Config(obj.(*Config)) })
	scheme.AddTypeDefaultingFunc(&ConfigList{}, func(obj interface{}) { SetObjectDefaults_ConfigList(obj.(*ConfigList)) })
	scheme.AddTypeDefaultingFunc(&Environment{}, func(obj interface{}) { SetObjectDefaults_Environment(obj.(*Environment)) })
	scheme.AddTypeDefaultingFunc(&EnvironmentList{}, func(obj interface{}) { SetObjectDefaults_EnvironmentList(obj.(*EnvironmentList)) })
	scheme.AddTypeDefaultingFunc(&Provider{}, func(obj interface{}) { SetObjectDefaults_Provider(obj.(*Provider)) })
	scheme.AddTypeDefaultingFunc(&ProviderList{}, func(obj interface{}) { SetObjectDefaults_ProviderList(obj.(*ProviderList)) })
	return nil
}

func SetObjectDefaults_Config(in *Config) {
	SetDefaults_Config(in)
	for i := range in.Clusters {
		a := &in.Clusters[i]
		clusterv1alpha1.SetDefaults_Cluster(a)
		for j := range a.InstancePools {
			b := &a.InstancePools[j]
			clusterv1alpha1.SetDefaults_InstancePool(b)
			for k := range b.Volumes {
				c := &b.Volumes[k]
				clusterv1alpha1.SetDefaults_Volume(c)
			}
		}
		if a.Kubernetes != nil {
			if a.Kubernetes.ClusterAutoscaler != nil {
				clusterv1alpha1.SetDefaults_ClusterKubernetesClusterAutoscaler(a.Kubernetes.ClusterAutoscaler)
			}
			if a.Kubernetes.APIServer != nil {
				if a.Kubernetes.APIServer.Amazon != nil {
					if a.Kubernetes.APIServer.Amazon.PublicELBAccessLogs != nil {
						clusterv1alpha1.SetDefaults_ClusterKubernetesAPIServerAmazonAccessLogs(a.Kubernetes.APIServer.Amazon.PublicELBAccessLogs)
					}
					if a.Kubernetes.APIServer.Amazon.InternalELBAccessLogs != nil {
						clusterv1alpha1.SetDefaults_ClusterKubernetesAPIServerAmazonAccessLogs(a.Kubernetes.APIServer.Amazon.InternalELBAccessLogs)
					}
				}
			}
		}
	}
	for i := range in.Providers {
		a := &in.Providers[i]
		SetObjectDefaults_Provider(a)
	}
	for i := range in.Environments {
		a := &in.Environments[i]
		SetObjectDefaults_Environment(a)
	}
}

func SetObjectDefaults_ConfigList(in *ConfigList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_Config(a)
	}
}

func SetObjectDefaults_Environment(in *Environment) {
	SetDefaults_Environment(in)
}

func SetObjectDefaults_EnvironmentList(in *EnvironmentList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_Environment(a)
	}
}

func SetObjectDefaults_Provider(in *Provider) {
	SetDefaults_Provider(in)
}

func SetObjectDefaults_ProviderList(in *ProviderList) {
	for i := range in.Items {
		a := &in.Items[i]
		SetObjectDefaults_Provider(a)
	}
}
