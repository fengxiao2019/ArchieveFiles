package restore

import (
	"fmt"

	"github.com/linxGnu/grocksdb"
)

// RestoreBackupToPlain restores a BackupEngine format backup to a plain RocksDB directory
func RestoreBackupToPlain(backupDir, restoreDir string) error {
	opts := grocksdb.NewDefaultOptions()
	defer opts.Destroy()

	backupEngine, err := grocksdb.OpenBackupEngine(opts, backupDir)
	if err != nil {
		return fmt.Errorf("failed to open backup engine: %v", err)
	}
	defer backupEngine.Close()

	restoreOpts := grocksdb.NewRestoreOptions()
	defer restoreOpts.Destroy()

	// Restore latest backup
	err = backupEngine.RestoreDBFromLatestBackup(restoreDir, restoreDir, restoreOpts)
	if err != nil {
		return fmt.Errorf("failed to restore backup: %v", err)
	}

	return nil
}
