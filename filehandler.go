package tinygit

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func generateVersionFile(rootPath string, v *Versioning) error {
	dirVerision := filepath.Join(rootPath, versionDirName)
	if _, err := os.Stat(dirVerision); os.IsNotExist(err) {
		err := generateVersionDir(rootPath)
		if err != nil {
			fmt.Println("Erro ao criar diretório de versão:", err)
			return err
		}
	}

	versionJson, err := json.MarshalIndent(v, "", "  ")

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

func decompressVersionFile(rootPath string) (*Versioning, error) {
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
	var v Versioning
	err = json.Unmarshal(buf.Bytes(), &v)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar o JSON: %v", err)
	}

	return &v, nil
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
func readDir(rootPath, dirPath string, ext, ignore *[]string) ([]*Node, error) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var children []*Node
	for _, entry := range dirEntries {
		if !entry.IsDir() && !contains(*ext, filepath.Ext(entry.Name())) || entry.Name() == versionDirName || contains(*ignore, entry.Name()) {
			continue
		}
		childPath := filepath.Join(dirPath, entry.Name())
		childNode, err := buildTree(rootPath, childPath, ext, ignore)
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

func CompressFilesToSend(c Changes, rootPath string) ([]bytes.Buffer, error) {
	var files []string
	for _, node := range c.Added {
		if node.Type == treeType {
			continue
		}
		files = append(files, node.Path)
	}
	for _, node := range c.Modified {
		if node.Type == treeType {
			continue
		}
		files = append(files, node.Path)
	}

	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)
	for _, file := range files {
		fileInfo, err := os.Stat(file)
		if err != nil {
			return nil, err
		}
		file, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		header, err := zip.FileInfoHeader(fileInfo)
		if err != nil {
			return nil, err
		}

		header.Name = filepath.Base(file.Name())
		header.Method = zip.Deflate
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(writer, file)
		if err != nil {
			return nil, err
		}
	}
	err := zipWriter.Close()
	if err != nil {
		return nil, err
	}

	return []bytes.Buffer{buf}, nil
}

func addFileToZip(zipWriter *zip.Writer, path, relPath string, info os.FileInfo) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = relPath
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}
