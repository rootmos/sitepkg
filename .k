export SITEPKG_LOG_LEVEL=DEBUG
#export SITEPKG_LOG_LEVEL=INFO

run() {
    make test
}

run2() {
    #make run \
        #SITEPKG_MANIFEST=example.manifest \
        #SITEPKG_CREATE=s3://rootmos-config/example.tar
    #make run \
        #SITEPKG_MANIFEST=example.manifest \
        #SITEPKG_CHROOT=tmp \
        #SITEPKG_EXTRACT=s3://rootmos-config/example.tar

    #make run \
        #SITEPKG_MANIFEST=example.manifest \
        #SITEPKG_CREATE=file://example.tar
    #make run \
        #SITEPKG_MANIFEST=example.manifest \
        #SITEPKG_CHROOT=tmp \
        #SITEPKG_EXTRACT=file://example.tar

    make run \
        SITEPKG_MANIFEST=example.manifest \
        SITEPKG_CHROOT=tmp \
        SITEPKG_EXTRACT=s3://rootmos-config/oops.tar \
        SITEPKG_NO_EXISTS_OK=1
}
