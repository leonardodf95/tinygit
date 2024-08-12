//go:build windows

package tinygit

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func generateVersionDir(rootPath string) error {
	dirVerision := filepath.Join(rootPath, versionDirName)

	if _, err := os.Stat(dirVerision); os.IsNotExist(err) {
		err := os.MkdirAll(dirVerision, 0700)
		if err != nil {
			fmt.Println("Erro ao criar diretório de versão:", err)
			return err
		}

		dirVerisionPtr, err := syscall.UTF16PtrFromString(dirVerision)
		if err != nil {
			fmt.Println("Erro ao converter o caminho para UTF16:", err)
			return err
		}
		err = syscall.SetFileAttributes(dirVerisionPtr, syscall.FILE_ATTRIBUTE_HIDDEN)
		if err != nil {
			fmt.Println("Erro ao ocultar o diretório de versão:", err)
			return err
		}
	}

	return nil
}
