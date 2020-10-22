# download and extract kubernets binaries
class kubernetes::install{
  include kubernetes

  $archive_url = "https://dl.k8s.io/v${::kubernetes::version}/kubernetes-${::kubernetes::release_type}-${::kubernetes::os_release}-${::kubernetes::release_arch}.tar.gz"
  $post_1_17 = versioncmp($::kubernetes::version, '1.17.0') >= 0

  if $post_1_17 {
      $checksum_type = 'sha512'
      $checksum_url = "${archive_url}.sha512"
  } else {
      $checksum_type = 'sha1'
      $checksum_url = "${archive_url}.sha1"
  }
  
  file { $::kubernetes::_dest_dir:
    ensure => 'directory',
    owner  => 'root',
    group  => 'root',
    mode   => '0755',
  }

  archive { "${::kubernetes::_dest_dir}.tar.gz":
    path            => "/tmp/${::kubernetes::_dest_dir}.tar.gz",
    source          => $archive_url,
    checksum_url    => $checksum_url,
    checksum_type   => $checksum_type,
    checksum_verify => true,
    extract         => true,
    extract_path    => $::kubernetes::_dest_dir,
    creates         => "${::kubernetes::_dest_dir}/kubernetes",
    cleanup         => true,
    require         => File[$::kubernetes::_dest_dir],
  }

  $::kubernetes::binaries.each |String $binary| {
    file { "${::kubernetes::_dest_dir}/${binary}":
      ensure => 'present',
      owner  => 'root',
      group  => 'root',
      mode   => '0755',
      source => "${::kubernetes::_dest_dir}/kubernetes/${::kubernetes::release_type}/bin/${binary}",
    }
  }

  tidy { '/opt':
   recurse  => true,
   matches  => [ '!($::kubernetes::_dest_dir)', 'kubernetes-*'],
   rmdirs   => true,
  }
}
