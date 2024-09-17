package tinygit

import (
	"fmt"
	"os"
)

// Structure to represent the versioning file
type Versioning struct {
	ExtensionsToGenerateVersion []string
	ignoredFiles                []string
	Head                        string
	Tree                        Node
}

// Structure to represent the tree of files and directories
type Node struct {
	Path     string  `json:"path"`
	Hash     string  `json:"hash"`
	Type     string  `json:"type"` // "blob" para arquivos, "tree" para diretórios
	Children []*Node `json:"children,omitempty"`
}

// Structure to represent the changes between two trees
type Changes struct {
	Added    []*Node `json:"added"`
	Removed  []*Node `json:"removed"`
	Modified []*Node `json:"modified"`
}

const (
	versionFileName = "version"
	versionDirName  = ".tinygit"
	blobType        = "blob"
	treeType        = "tree"
)

// Function to initialize the version control in the directory
func InitControlVersion(path string, extPermited, ignoredFiles []string) error {
	if VerifyIfExistVersionControl(path) {
		fmt.Println("Controle de versão já inicializado.")
		return nil
	}
	err := generateVersionDir(path)
	if err != nil {
		fmt.Println("Erro ao criar diretório de versão:", err)
		return err
	}

	fmt.Println("Controle de versão inicializado em", path)
	tree, err := buildTree(path, path, &extPermited, &ignoredFiles)
	if err != nil {
		fmt.Println("Erro ao construir a árvore:", err)
		return err
	}
	if tree == nil {
		fmt.Println("A árvore está vazia.")
		tree = &Node{
			Hash: "",
		}
	}

	v := Versioning{
		ExtensionsToGenerateVersion: extPermited,
		ignoredFiles:                ignoredFiles,
		Head:                        tree.Hash,
		Tree:                        *tree,
	}

	err = generateVersionFile(path, &v)
	if err != nil {
		fmt.Println("Erro ao salvar a árvore:", err)
		return err
	}
	return nil
}

func CommitControlVersion(path string, ext, ignore []string) error {
	if ignore == nil {
		ignore = []string{}
	}

	c, v, err := StatusControlVersion(path, ext, ignore)
	if err != nil {
		fmt.Println("Erro ao verificar o status:", err)
		return err
	}

	if c == nil {
		return nil
	}
	if len(c.Modified) == 0 && len(c.Added) == 0 && len(c.Removed) == 0 {
		return nil
	}

	v.Tree = *c.Modified[0]

	fmt.Println("Salvando árvore de versionamento...")
	err = generateVersionFile(path, v)
	if err != nil {
		fmt.Println("Erro ao salvar a árvore:", err)
		return err
	}

	return nil
}

func StatusControlVersion(path string, ext, ignore []string) (*Changes, *Versioning, error) {
	fmt.Println("Verificando status de controle de versão em", path)
	if !VerifyIfExistVersionControl(path) {
		return nil, nil, fmt.Errorf("controle de versão não inicializado")
	}

	v, err := decompressVersionFile(path)
	if err != nil {
		fmt.Println("Erro ao ler a árvore salva:", err)
		return nil, nil, err
	}

	if ignore != nil {
		diff := CompareSlices(v.ignoredFiles, ignore)
		if len(diff) > 0 {
			v.ignoredFiles = append(v.ignoredFiles, diff...)
		}
	}

	if ext != nil {
		diff := CompareSlices(v.ExtensionsToGenerateVersion, ext)
		if len(diff) > 0 {
			v.ExtensionsToGenerateVersion = append(v.ExtensionsToGenerateVersion, diff...)
		}
	}

	currentTree, err := buildTree(path, path, &v.ExtensionsToGenerateVersion, &v.ignoredFiles)
	if err != nil {
		fmt.Println("Erro ao construir a árvore:", err)
		return nil, nil, err
	}

	changes := CompareTrees(&v.Tree, currentTree)

	if len(changes.Modified) == 0 && len(changes.Added) == 0 && len(changes.Removed) == 0 {
		fmt.Println("Nenhuma mudança detectada.")
		return nil, nil, nil
	}

	if len(changes.Modified) > 0 {
		m := []Node{}
		for _, modified := range changes.Modified {
			if modified.Type == treeType {
				continue
			}
			m = append(m, *modified)
		}
		if len(m) > 0 {
			fmt.Println("Modificados:")
			for _, modified := range m {
				fmt.Println(modified.Path)
			}
		}
	}
	if len(changes.Added) > 0 {
		a := []Node{}
		for _, added := range changes.Added {
			if added.Type == treeType {
				continue
			}
			a = append(a, *added)
		}
		if len(a) > 0 {
			fmt.Println("Adicionados:")
			for _, added := range a {
				fmt.Println(added.Path)
			}
		}
	}
	if len(changes.Removed) > 0 {
		r := []Node{}
		for _, removed := range changes.Removed {
			if removed.Type == treeType {
				continue
			}
			r = append(r, *removed)
		}
		if len(r) > 0 {

			fmt.Println("Removidos:")
			for _, removed := range r {
				fmt.Println(removed.Path)
			}
		}
	}

	return changes, v, nil
}

func PrintVersionFile(path string) error {
	v, err := decompressVersionFile(path)
	if err != nil {
		fmt.Println("Erro ao ler a árvore salva:", err)
		return err
	}
	fileTree, err := os.OpenFile("tree.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		fmt.Println("Erro ao abrir o arquivo:", err)
		return err
	}
	fileTree.WriteString("HEAD: " + v.Head + "\n")
	fileTree.WriteString("Árvore:\n")
	fileTree.Close()
	fmt.Println("HEAD:", v.Head)
	fmt.Println("Árvore:")
	printTree(&v.Tree)
	return nil
}

func CloneRepository(path string, server string, params map[string]string) error {
	if VerifyIfExistVersionControl(path) {
		fmt.Println("Controle de versão já inicializado.")
		return nil
	}
	err := generateVersionDir(path)
	if err != nil {
		fmt.Println("Erro ao criar diretório de versão:", err)
		return err
	}
	fmt.Println("Controle de versão inicializado em", path)
	fmt.Println("Clonando repositório...")

	v, err := RequestClone(path, server, params)

	if err != nil {
		fmt.Println("Erro ao clonar o repositório:", err)
		return err
	}

	if v == nil {
		return fmt.Errorf("extensões não informadas")
	}

	fmt.Println("Repositório clonado, gerando árvore de versionamento...")
	tree, err := buildTree(path, path, &v.ExtensionsToGenerateVersion, &v.ignoredFiles)
	if err != nil {
		fmt.Println("Erro ao construir a árvore:", err)
		return err
	}
	if tree == nil {
		fmt.Println("A árvore está vazia.")
		return nil
	}

	v.Head = tree.Hash
	v.Tree = *tree

	fmt.Println("Salvando árvore de versionamento...")
	err = generateVersionFile(path, v)
	if err != nil {
		fmt.Println("Erro ao salvar a árvore:", err)
		return err
	}
	return nil
}

func PullRepository(path string, server string, parameter map[string]string) error {

	if !VerifyIfExistVersionControl(path) {
		return fmt.Errorf("controle de versão não inicializado")
	}

	fmt.Println("Atualizando repositório...")
	vCurrent, err := decompressVersionFile(path)

	if err != nil {
		fmt.Println("Erro ao ler a árvore salva:", err)
		return fmt.Errorf("erro ao ler a árvore salva: %w", err)
	}

	fmt.Println("Enviando HEAD para o servidor... " + vCurrent.Head)
	hasModifications := sendHeadOfVersion(vCurrent.Head, server, parameter)

	if !hasModifications {
		fmt.Println("Repositório já está atualizado.")
		return nil
	}

	fmt.Println("Repositório atualizado, gerando árvore de versionamento...")
	err = sendTreeOfVersionForUpdate(path, &vCurrent.Tree, server, parameter)
	if err != nil {
		fmt.Println("Erro ao enviar a árvore:", err)
		return fmt.Errorf("erro ao enviar a árvore: %w", err)
	}

	fmt.Println("Árvore de versionamento atualizada, gerando árvore local...")
	tree, err := buildTree(path, path, &vCurrent.ExtensionsToGenerateVersion, &vCurrent.ignoredFiles)
	if err != nil {
		fmt.Println("Erro ao construir a árvore:", err)
		return fmt.Errorf("erro ao construir a árvore: %w", err)
	}
	if tree == nil {
		fmt.Println("A árvore está vazia.")
		return nil
	}

	vCurrent.Head = tree.Hash
	vCurrent.Tree = *tree

	fmt.Println("Salvando árvore de versionamento...")
	err = generateVersionFile(path, vCurrent)
	if err != nil {
		fmt.Println("Erro ao salvar a árvore:", err)
		return fmt.Errorf("erro ao salvar a árvore: %w", err)
	}
	return nil
}

func PushRepository(path string, server string, parameters map[string]string) error {
	if !VerifyIfExistVersionControl(path) {
		return fmt.Errorf("controle de versão não inicializado")
	}

	fmt.Println("Enviando HEAD para o servidor...")
	vCurrent, err := decompressVersionFile(path)

	if err != nil {
		fmt.Println("Erro ao ler a árvore salva:", err)
		return fmt.Errorf("erro ao ler a árvore salva: %w", err)
	}

	hasModifications := sendHeadOfVersion(vCurrent.Head, server, parameters)
	if !hasModifications {
		fmt.Println("Repositório já está atualizado.")
		return nil
	}

	fmt.Println("Repositório atualizado, enviando árvore de versionamento...")
	c, err := sendTreeOfVersion(&vCurrent.Tree, server, parameters)

	if err != nil {
		fmt.Println("Erro ao enviar a árvore:", err)
		return fmt.Errorf("erro ao enviar a árvore: %w", err)
	}

	if c == nil {
		fmt.Println("Árvore de versionamento já está atualizada.")
		return nil
	}

	buf, err := CompressFilesToSend(*c, path)

	if err != nil {
		fmt.Println("Erro ao compactar os arquivos:", err)
		return fmt.Errorf("erro ao compactar os arquivos: %w", err)
	}

	err = sendFilesToServer(buf, server, parameters)

	if err != nil {
		fmt.Println("Erro ao enviar os arquivos:", err)
		return fmt.Errorf("erro ao enviar os arquivos: %w", err)
	}

	fmt.Println("Arquivos enviados com sucesso.")
	return nil
}

func GetTreeControlVersion(path string) (*Node, error) {
	v, err := decompressVersionFile(path)
	if err != nil {
		fmt.Println("Erro ao ler a árvore salva:", err)
		return nil, err
	}
	return &v.Tree, nil
}
