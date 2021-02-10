class site_module::docker{
  $service_name = 'docker'
  $path = defined('$::path') ? {
      default => '/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/opt/bin',
      true    => $::path
  }

  $file_path = '/root/new-docker'
  $file_exists = find_file($file_path)

  if $file_exists  {
    package{'docker-ce': ensure  => 'present'} 
    package{'docker-ce-cli': ensure  => 'present'} 
    package{'containerd.io': ensure  => 'present'} 
    -> class{'site_module::docker_ce_config':}
    ~> exec { "${service_name}-daemon-reload":
      command     => 'systemctl daemon-reload',
      path        => $path,
      refreshonly => true,
    }
    -> service{"${service_name}.service":
      ensure     => running,
      enable     => true,
      hasstatus  => true,
      hasrestart => true,
    }

    if defined(Class['kubernetes::kubelet']){
      Class['site_module::docker'] -> Class['kubernetes::kubelet']
    }

    if defined(Class['prometheus']){
      Class['site_module::docker'] -> Class['prometheus']
    }
  } else {
    package{'docker':
      ensure  => present,
    }
    -> class{'site_module::docker_config':}
    ~> exec { "${service_name}-daemon-reload":
      command     => 'systemctl daemon-reload',
      path        => $path,
      refreshonly => true,
    }
    -> service{"${service_name}.service":
      ensure     => running,
      enable     => true,
      hasstatus  => true,
      hasrestart => true,
    }

    if defined(Class['kubernetes::kubelet']){
      Class['site_module::docker'] -> Class['kubernetes::kubelet']
    }

    if defined(Class['prometheus']){
      Class['site_module::docker'] -> Class['prometheus']
    }
  }
}

