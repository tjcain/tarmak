class site_module::docker_ce_config {
  include ::aws_ebs

  $disks = aws_ebs::disks()

  case $disks.length {
    0: {$ebs_device = undef}
    1: {$ebs_device = $disks[0]}
    default: {$ebs_device = $disks[1]}
  }

  file { "/etc/systemd/system/${::site_module::docker::service_name}.service.d":
    ensure  => absent,
    recurse => true,
    force   => true,
  } -> file { '/etc/sysconfig/docker-storage-setup':
    ensure  => absent,
    force => true,
  } -> file { "/etc/docker":
    ensure  => directory,
  } -> file { '/etc/docker/daemon.json':
    ensure  => file,
    content => template('site_module/docker-config.erb'),
    notify  => Service["${::site_module::docker::service_name}.service"],
  }
}
