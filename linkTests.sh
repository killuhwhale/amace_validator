# Mkdir if DNE $CHROMEOS_SRC/src/platform/tast-tests/src/chromiumos/tast/local/bundles/cros/arc/amace/
directory="$CHROMEOS_SRC/src/platform/tast-tests/src/chromiumos/tast/local/bundles/cros/arc/amace/"
if [ ! -d "$directory" ]; then
    echo "Creating directory: $directory"
    mkdir -p "$directory"
else
    echo "Directory already exists: $directory"
fi


# Data files
ln -s $CHROMEOS_SRC/src/platform/tast-tests/src/chromiumos/tast/local/bundles/cros/arc/data/AMACE_app_list.tsv  ./AMACE_app_list.tsv
# ln -s $CHROMEOS_SRC/src/platform/tast-tests/src/chromiumos/tast/local/bundles/cros/arc/data/AMACE_secret.txt  ./AMACE_secret.tsv

# Helper Functions
ln -s $CHROMEOS_SRC/src/platform/tast-tests/src/chromiumos/tast/local/bundles/cros/arc/amace/deviceutils.go ./amace/deviceutils.go
ln -s $CHROMEOS_SRC/src/platform/tast-tests/src/chromiumos/tast/local/bundles/cros/arc/amace/dismissMiscProps.go ./amace/dismissMiscProps.go
ln -s $CHROMEOS_SRC/src/platform/tast-tests/src/chromiumos/tast/local/bundles/cros/arc/amace/installAppUtils.go ./amace/installAppUtils.go
ln -s $CHROMEOS_SRC/src/platform/tast-tests/src/chromiumos/tast/local/bundles/cros/arc/amace/loadFiles.go ./amace/loadFiles.go
ln -s $CHROMEOS_SRC/src/platform/tast-tests/src/chromiumos/tast/local/bundles/cros/arc/amace/types.go ./amace/types.go


# Main Test
ln -s $CHROMEOS_SRC/src/platform/tast-tests/src/chromiumos/tast/local/bundles/cros/arc/amace.go ./amace.go

