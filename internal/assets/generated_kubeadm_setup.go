// Code generated automatically with go generate; DO NOT EDIT.

package assets

const KubeadmSetupScriptCode = `#!/bin/sh

##########################################################################################
# kubeadm setup script
##########################################################################################

LSB_RELEASE=/usr/bin/lsb_release

# the executable that packages will install, and the packages per distro
KUBEADM_EXE=/usr/bin/kubeadm

KUBEADM_PKG_SUSE="kubernetes-kubeadm"
KUBEADM_PKG_SUSE_REPO="https://download.opensuse.org/repositories/devel:/kubic/openSUSE_Leap_15.1/"
KUBEADM_PKG_SUSE_REPOFILE="/etc/zypp/repos.d/kubernetes.repo"
KUBEADM_PKG_SUSE_PACKAGES="$KUBEADM_PKG_SUSE kubernetes-kubelet kubernetes-client"

KUBEADM_PKG_APT="kubeadm"
KUBEADM_PKG_APT_REPO="http://apt.kubernetes.io/"
KUBEADM_PKG_APT_GPG="https://packages.cloud.google.com/apt/doc/apt-key.gpg"
KUBEADM_PKG_APT_PACKAGES="$KUBEADM_PKG_APT kubelet kubectl docker.io kubernetes-cni"
KUBEADM_PKG_APT_PACKAGES_PRE="apt-transport-https ebtables ethtool"
KUBEADM_PKG_APT_SRCLST="/etc/apt/sources.list.d/kubernetes.list"

KUBEADM_PKG_YUM="kubeadm"
KUBEADM_PKG_YUM_REPOFILE="/etc/yum.repos.d/kubernetes.repo"
KUBEADM_PKG_YUM_PACKAGES="$KUBEADM_PKG_YUM kubelet kubernetes-cni docker kubectl"

# we will try to discover the DIST and RELEASE
ID=
DIST=
RELEASE=

##########################################################################################

log()    { echo "[kubeadm setup] $@" ;     }
abort()  { log "$@" ; exit 1 ; }

restart_services() {
    log "starting services"
    systemctl enable docker   && systemctl start docker  || abort "could not start docker"
    systemctl enable kubelet  && systemctl start kubelet || abort "could not start kubelet"
}

##########################################################################################

# installation for SUSE variants: OpenSUSE/SLE/CaaSP...
install_zypper() {
    source /etc/os-release

	if [ ! -f $KUBEADM_PKG_SUSE_REPOFILE ] ; then
		local name=$(echo $NAME | tr " " "_")
		local ver=$VERSION

		cat <<EOF > $KUBEADM_PKG_SUSE_REPOFILE
[kubernetes]
name=Kubernetes
baseurl=https://download.opensuse.org/repositories/devel:/kubic/$name_$ver/
enabled=1
gpgcheck=1
repo_gpgcheck=1
EOF
		zypper refresh
	else
		log "repository already found: skipping installation of the repo"
	fi

	log "checking we have everything we need..."
	zypper $zypper_args in -y $KUBEADM_PKG_SUSE_PACKAGES || abort "could not finish the installation of packages"
	log "... everything installed"
	restart_services
}

# installation for RedHat variants: RedHat/CentOS...
install_yum() {
	log "Installing for RedHat..."
	if [ ! -f $KUBEADM_PKG_YUM_REPOFILE ] ; then
		[ -n "$RELEASE" ] || abort "can't continue without knowing the release"
		cat <<EOF > $KUBEADM_PKG_YUM_REPOFILE
[kubernetes]
name=Kubernetes
baseurl=http://yum.kubernetes.io/repos/kubernetes-el$RELEASE-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg
       https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF
		setenforce 0
	else
		log "repository already found: skipping installation of the repo"
	fi

    log "checking we have everything we need..."
    [ -x $KUBEADM_EXE ] || yum install -y $KUBEADM_PKG_YUM_PACKAGES || abort "could not finish the installation of kubeadm"
    log "... everything installed"
	restart_services
}

# installation for Debian variants: debian/Ubuntu...
install_apt() {
	log "installing for Ubuntu|Debian..."
	if [ ! -f $KUBEADM_PKG_APT_SRCLST ] ; then
		apt-get update && apt-get install -y $KUBEADM_PKG_APT_PACKAGES_PRE
		curl -s "$KUBEADM_PKG_APT_GPG" | apt-key add -
		echo "deb $KUBEADM_PKG_APT_REPO kubernetes-xenial main" >> $KUBEADM_PKG_APT_SRCLST
		apt-get update
	else
		log "repository already found: skipping installation of the repo"
	fi

    log "checking we have everything we need..."
    [ -x $KUBEADM_EXE ] || apt-get install -y $KUBEADM_PKG_APT_PACKAGES || abort "could not finish the installation of kubectl"
    log "... everything installed"
	restart_services
}

##########################################################################################

# there are two ways we can identify the distro: with the help of lsb-release, or
# with some key files in /etc (like /etc/debian_version)
if [ -x $LSB_RELEASE ] ; then
	ID=$($LSB_RELEASE --short --id)
	case $ID in
	RedHatEnterpriseServer|CentOS|Fedora)
		RELEASE=$($LSB_RELEASE --short --release | cut -d. -f1)
		install_yum
		;;

	Ubuntu|Debian)
		install_apt
		;;

	*SUSE*)
		desc=$($LSB_RELEASE --short --description)
		RELEASE=$($LSB_RELEASE --short --release)
		case $desc in
		*openSUSE*)
		    DIST=opensuse$RELEASE
		    ;;
		*Enterprise*)
		    DIST=sles$RELEASE
		    ;;
		esac
		install_zypper
		;;

	*)
		log "could not get the release/distribution from $LSB_RELEASE"
		DIST=
		;;
	esac
else
	if [ -f /etc/debian_version ] ; then
		install_apt
	elif [ -f /etc/fedora-release ] ; then
		install_yum
	elif [ -f /etc/redhat-release ] ; then
		install_yum
	elif [ -f /etc/SuSE-release ] ; then
		install_zypper
	elif [ -f /etc/centos-release ] ; then
		install_yum
	fi
fi

[ -x $KUBEADM_EXE ] || abort "no kubeadm executable available at $KUBEADM_EXE"



`