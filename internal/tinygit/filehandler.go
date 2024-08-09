package tinygit

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func generateVersionFile(rootPath string, tree Node) error {
	dirVerision := filepath.Join(rootPath, versionDirName)
	if _, err := os.Stat(dirVerision); os.IsNotExist(err) {
		err := generateVersionDir(rootPath)
		if err != nil {
			fmt.Println("Erro ao criar diretório de versão:", err)
			return err
		}
	}

	versionJson, err := json.MarshalIndent(tree, "", "  ")

	if err != nil {
		fmt.Println("Erro ao codificar a árvore em JSON:", err)
		return err
	}

	err = compressVersionFile(dirVerision, versionJson)

	if err != nil {
		fmt.Println("Erro ao compactar o arquivo de versão:", err)
		return err
	}

	return nil
}

// Compacta o arquivo JSON em um arquivo ZIP
func compressVersionFile(rootPath string, content []byte) error {
	fileVersion := filepath.Join(rootPath, versionFileName)

	// Cria o arquivo compactado
	compressedFile, err := os.Create(fileVersion)
	if err != nil {
		return fmt.Errorf("erro ao criar o arquivo compactado: %v", err)
	}
	defer compressedFile.Close()

	// Escreve o conteúdo JSON no arquivo compactado
	writer := gzip.NewWriter(compressedFile)
	defer writer.Close()

	_, err = writer.Write(content)
	if err != nil {
		return fmt.Errorf("erro ao escrever o conteúdo no arquivo compactado: %v", err)
	}
	return nil
}

func decompressVersionFile(rootPath string) (*Node, error) {
	dirVerision := filepath.Join(rootPath, versionDirName)
	fileVersion := filepath.Join(dirVerision, versionFileName)

	// Abre o arquivo compactado
	compressedFile, err := os.Open(fileVersion)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir o arquivo compactado: %v", err)
	}
	defer compressedFile.Close()

	// Descompacta o arquivo
	reader, err := gzip.NewReader(compressedFile)
	if err != nil {
		return nil, fmt.Errorf("erro ao descompactar o arquivo: %v", err)
	}
	defer reader.Close()

	// salva os dados em um buffer
	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return nil, fmt.Errorf("erro ao copiar os dados para o buffer: %v", err)
	}

	// Decodifica o JSON
	var tree Node
	err = json.Unmarshal(buf.Bytes(), &tree)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar o JSON: %v", err)
	}

	return &tree, nil
}

func contains(slice []string, element string) bool {
	for _, item := range slice {
		if strings.EqualFold(item, element) {
			return true
		}
	}
	return false
}

// Lê um diretório e retorna os nós filhos
func readDir(dirPath string) ([]*Node, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var children []*Node
	for _, entry := range dirEntries {
		if !entry.IsDir() && !contains(acceptedExtensions, filepath.Ext(entry.Name())) || entry.Name() == versionDirName {
			continue
		}
		childPath := filepath.Join(dirPath, entry.Name())
		childNode, err := buildTree(childPath)
		if err != nil {
			return nil, err
		}
		if childNode == nil {
			continue
		}
		children = append(children, childNode)
	}

	return children, nil
}

func VerifyIfExistVersionControl(path string) bool {
	dirVerision := filepath.Join(path, versionDirName)
	fileVersion := filepath.Join(dirVerision, versionFileName)

	if _, err := os.Stat(dirVerision); os.IsNotExist(err) {
		return false
	}

	if _, err := os.Stat(fileVersion); os.IsNotExist(err) {
		return false
	}

	return true
}
