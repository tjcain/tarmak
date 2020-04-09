class kubernetes_addons::node_label_bodger(
  Optional[String] $version='',
  String $image='',
) inherits ::kubernetes_addons::params {
  require ::kubernetes

  kubernetes::apply{'node-label-bodger':
    ensure    => $ensure,
    manifests => [
      template('kubernetes_addons/node-label-bodger.yaml.erb'),
    ],
  }
}
