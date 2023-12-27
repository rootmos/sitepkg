export SITEPKG_LOG_LEVEL=DEBUG
#export SITEPKG_LOG_LEVEL=INFO
#export SITEPKG_LOG_JSON=1
#export SITEPKG_GZIP=9
export SITEPKG_KEYFILE=$K_DIR/.keyfile

new-keyfile() {
    head -c32 /dev/random > .keyfile
}

run0() {
    make test VERBOSE=1
}

run() {
    #make run \
        #SITEPKG_MANIFEST=example.manifest \
        #SITEPKG_CREATE=s3://rootmos-config/example.tar
    #make run \
        #SITEPKG_MANIFEST=example.manifest \
        #SITEPKG_CHROOT=tmp \
        #SITEPKG_EXTRACT=s3://rootmos-config/example.tar

    make run \
        SITEPKG_MANIFEST=example.manifest \
        SITEPKG_CREATE=file://example.tar.gz.enc
    make run \
        SITEPKG_MANIFEST=example.manifest \
        SITEPKG_CHROOT=tmp \
        SITEPKG_EXTRACT=file://example.tar.gz.enc

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
