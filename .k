export SITEPKG_LOG_LEVEL=DEBUG
#export SITEPKG_LOG_LEVEL=INFO
#export SITEPKG_LOG_JSON=1

run() {
    make test VERBOSE=1
}

run0() {
    make run \
        SITEPKG_MANIFEST=example.manifest \
        SITEPKG_CREATE=s3://rootmos-config/example.tar
    make run \
        SITEPKG_MANIFEST=example.manifest \
        SITEPKG_CHROOT=tmp \
        SITEPKG_EXTRACT=s3://rootmos-config/example.tar

    #make run \
        #SITEPKG_MANIFEST=example.manifest \
        #SITEPKG_CREATE=file://example.tar
    #make run \
        #SITEPKG_MANIFEST=example.manifest \
        #SITEPKG_CHROOT=tmp \
        #SITEPKG_EXTRACT=file://example.tar

    #make run \
        #SITEPKG_MANIFEST=example.manifest \
        #SITEPKG_CHROOT=tmp \
        #SITEPKG_EXTRACT=s3://rootmos-config/oops.tar \
        #SITEPKG_NO_EXISTS_OK=1

    true
}

docker() {
    make docker SUDO="sudo -A"
}
