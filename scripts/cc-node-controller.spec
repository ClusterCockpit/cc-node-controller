Name:           cc-node-controller
Version:        %{VERS}
Release:        1%{?dist}
Summary:        Node controller daemon from the ClusterCockpit suite

License:        MIT
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  go-toolset
BuildRequires:  systemd-rpm-macros
# for header downloads
BuildRequires:  git
# Recommended when using the sysusers_create_package macro
#Requires(pre): /usr/bin/systemd-sysusers

Provides:       %{name} = %{version}

%description
Node controller daemon from the ClusterCockpit suite

%global debug_package %{nil}

%prep
%autosetup


%build
make


%install
install -Dpm 0750 %{name} %{buildroot}%{_bindir}/%{name}
install -Dpm 0600 config.json %{buildroot}%{_sysconfdir}/%{name}/%{name}.json


%check
# go test should be here... :)

%pre

%post

%preun

%files
# Binary
%attr(-,root,root) %{_bindir}/%{name}
# Config
%dir %{_sysconfdir}/%{name}
%attr(0600,root,root) %config(noreplace) %{_sysconfdir}/%{name}/%{name}.json

%changelog
* Tue Jul 22 2025 Michael Panzlaff - 0.1.0
- Initial release
