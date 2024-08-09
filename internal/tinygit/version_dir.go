//go:build linux || darwin

package tinygit

import (
	"fmt"
	"os"
	"path/filepath"
)

func generateVersionDir(rootPath string) error {
	dirVerision := filepath.Join(rootPath, versionDirName)

	if _, err := os.Stat(dirVerision); os.IsNotExist(err) {
		err := os.Mkdir(dirVerision, 0700)
		if err != nil {
			fmt.Println("Erro ao criar diretório de versão:", err)
			return err
		}
	}

	return nil
}
