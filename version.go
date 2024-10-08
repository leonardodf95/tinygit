package tinygit

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Constrói recursivamente a árvore
func buildTree(rootPath, path string, ext, ignore *[]string) (*Node, error) {
	// Calcula o caminho relativo em relação ao diretório base
	relativePath, err := filepath.Rel(rootPath, path)
	if err != nil {
		return nil, err
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	node := &Node{
		Path: relativePath,
	}

	if fileInfo.IsDir() {
		node.Type = treeType
		children, err := readDir(rootPath, path, ext, ignore)
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

func CompareTrees(savedNode, currentNode *Node) *Changes {
	changes := &Changes{}

	if (savedNode == nil || savedNode.Hash == "") && currentNode == nil {
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
			childChanges := CompareTrees(savedChild, currentChild)
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

func CompareHashes(hash1, hash2 string) bool {
	return strings.EqualFold(hash1, hash2)
}

func printTree(node *Node) {

	printFile, err := os.OpenFile("tree.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Erro ao abrir o arquivo:", err)
		return
	}
	defer printFile.Close()
	printFile.WriteString(fmt.Sprintf("%s (%s) - %s\n", node.Path, node.Type, node.Hash))
	fmt.Printf("%s (%s) - %s\n", node.Path, node.Type, node.Hash)
	for _, child := range node.Children {
		printTree(child)
	}
}

// Compara duas slices de strings e retorna os elementos diferentes que estão no slice2
func CompareSlices(slice1, slice2 []string) []string {
	diff := []string{}
	for _, s2 := range slice2 {
		found := false
		for _, s1 := range slice1 {
			if strings.EqualFold(s1, s2) {
				found = true
				break
			}
		}
		if !found {
			diff = append(diff, s2)
		}
	}
	return diff
}
