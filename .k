export SITEPKG_LOG_LEVEL=DEBUG

run() {
    make run \
        SITEPKG_MANIFEST=example.manifest \
        SITEPKG_OUTPUT=example.tar \
        && tar tf example.tar
}
