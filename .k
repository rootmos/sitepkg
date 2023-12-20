export SITEPKG_LOG_LEVEL=DEBUG

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
        SITEPKG_CREATE=file://example.tar
    make run \
        SITEPKG_MANIFEST=example.manifest \
        SITEPKG_CHROOT=tmp \
        SITEPKG_EXTRACT=file://example.tar
}
