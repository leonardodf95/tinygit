package tinygit

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"
)

// Constrói recursivamente a árvore
func buildTree(path string) (*Node, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	node := &Node{
		Path: path,
	}

	if fileInfo.IsDir() {
		node.Type = treeType
		children, err := readDir(path)
		if err != nil {
			return nil, err
		}
		if len(children) == 0 {
			return nil, nil
		}
		node.Children = children

		// Calcula o hash do diretório combinando os hashes dos filhos
		dirHash := sha1.New()
		for _, child := range children {
			io.WriteString(dirHash, child.Hash)
		}
		node.Hash = hex.EncodeToString(dirHash.Sum(nil))

	} else {
		node.Type = blobType
		node.Hash, err = calculateFileHash(path)
		if err != nil {
			return nil, err
		}
	}

	return node, nil
}

// Calcula o hash para um arquivo, incluindo seus metadados
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return "", err
	}

	// Lê o conteúdo do arquivo
	fileHash := sha1.New()
	if _, err := io.Copy(fileHash, file); err != nil {
		return "", err
	}
	contentHash := fileHash.Sum(nil)

	// Inclui metadados do arquivo no hash
	metaHash := sha1.New()
	metaData := fmt.Sprintf("%s%s%d", blobType, fileInfo.ModTime().Format(time.RFC3339), fileInfo.Size())
	metaHash.Write([]byte(metaData))
	metaHash.Write(contentHash)

	return hex.EncodeToString(metaHash.Sum(nil)), nil
}

func compareTrees(savedNode, currentNode *Node) *Changes {
	changes := &Changes{}

	if savedNode == nil && currentNode == nil {
		return changes
	}

	if savedNode == nil {
		changes.Added = append(changes.Added, currentNode)
		return changes
	}

	if currentNode == nil {
		changes.Removed = append(changes.Removed, savedNode)
		return changes
	}

	// Se os hashes são diferentes, marcamos o nó como modificado
	if savedNode.Hash != currentNode.Hash {
		changes.Modified = append(changes.Modified, currentNode)
	} else {
		// Se os hashes são iguais, não precisamos comparar os filhos
		return changes
	}

	// Mapear os filhos por caminho para comparação
	savedChildrenMap := make(map[string]*Node)
	for _, child := range savedNode.Children {
		savedChildrenMap[child.Path] = child
	}

	currentChildrenMap := make(map[string]*Node)
	for _, child := range currentNode.Children {
		currentChildrenMap[child.Path] = child
	}

	// Verifica por nós adicionados e modificados
	for path, currentChild := range currentChildrenMap {
		if savedChild, found := savedChildrenMap[path]; found {
			childChanges := compareTrees(savedChild, currentChild)
			changes.Added = append(changes.Added, childChanges.Added...)
			changes.Removed = append(changes.Removed, childChanges.Removed...)
			changes.Modified = append(changes.Modified, childChanges.Modified...)
		} else {
			changes.Added = append(changes.Added, currentChild)
		}
	}

	// Verifica por nós removidos
	for path, savedChild := range savedChildrenMap {
		if _, found := currentChildrenMap[path]; !found {
			changes.Removed = append(changes.Removed, savedChild)
		}
	}

	return changes
}
