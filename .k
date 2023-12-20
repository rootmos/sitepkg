export SITEPKG_LOG_LEVEL=INFO

run() {
    make run \
        SITEPKG_MANIFEST=example.manifest \
        SITEPKG_CREATE=example.tar \
        && tar tf example.tar

    make run \
        SITEPKG_MANIFEST=example.manifest \
        SITEPKG_CHROOT=tmp \
        SITEPKG_EXTRACT=example.tar
}
