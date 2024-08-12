package tinygit

import "fmt"

// Structure to represent the versioning file
type Versioning struct {
	ExtensionsToGenerateVersion []string
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

func InitControlVersion(path string, ext []string) error {
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
	tree, err := buildTree(path, &ext)
	if err != nil {
		fmt.Println("Erro ao construir a árvore:", err)
		return err
	}
	if tree == nil {
		fmt.Println("A árvore está vazia.")
		return nil
	}

	v := Versioning{
		ExtensionsToGenerateVersion: ext,
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

func CommitControlVersion(path string) error {
	c, err := StatusControlVersion(path)
	if err != nil {
		fmt.Println("Erro ao verificar o status:", err)
		return err
	}
	if c == nil {
		return nil
	}
	v, err := decompressVersionFile(path)
	if err != nil {
		fmt.Println("Erro ao ler a árvore salva:", err)
		return err
	}
	v.Tree = *c.Modified[0]

	err = generateVersionFile(path, v)
	if err != nil {
		fmt.Println("Erro ao salvar a árvore:", err)
		return err
	}

	return nil
}

func StatusControlVersion(path string) (*Changes, error) {
	fmt.Println("Verificando status de controle de versão em", path)
	if !VerifyIfExistVersionControl(path) {
		fmt.Println("Controle de versão não inicializado.")
		return nil, nil
	}

	v, err := decompressVersionFile(path)
	if err != nil {
		fmt.Println("Erro ao ler a árvore salva:", err)
		return nil, err
	}

	currentTree, err := buildTree(path, &v.ExtensionsToGenerateVersion)
	if err != nil {
		fmt.Println("Erro ao construir a árvore:", err)
		return nil, err
	}

	changes := CompareTrees(&v.Tree, currentTree)

	if len(changes.Modified) == 0 && len(changes.Added) == 0 && len(changes.Removed) == 0 {
		fmt.Println("Nenhuma mudança detectada.")
		return nil, nil
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

	return changes, nil
}

func PrintVersionFile(path string) error {
	v, err := decompressVersionFile(path)
	if err != nil {
		fmt.Println("Erro ao ler a árvore salva:", err)
		return err
	}

	fmt.Println("HEAD:", v.Head)
	fmt.Println("Árvore:")
	printTree(&v.Tree, 0)
	return nil
}

func CloneRepository(path string, server string) error {
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
	v, err := RequestClone(server, path)

	if err != nil {
		fmt.Println("Erro ao clonar o repositório:", err)
		return err
	}

	if v == nil {
		return fmt.Errorf("extensões não informadas")
	}

	fmt.Println("Repositório clonado, gerando árvore de versionamento...")
	tree, err := buildTree(path, &v.ExtensionsToGenerateVersion)
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

func PullRepository(path string, server string) error {

	if !VerifyIfExistVersionControl(path) {
		return fmt.Errorf("controle de versão não inicializado")
	}

	fmt.Println("Atualizando repositório...")
	vCurrent, err := decompressVersionFile(path)

	if err != nil {
		fmt.Println("Erro ao ler a árvore salva:", err)
		return fmt.Errorf("erro ao ler a árvore salva: %w", err)
	}

	hasModifications := sendHeadOfVersion(vCurrent.Head, server)

	if !hasModifications {
		fmt.Println("Repositório já está atualizado.")
		return nil
	}

	fmt.Println("Repositório atualizado, gerando árvore de versionamento...")
	err = sendTreeOfVersionForUpdate(&vCurrent.Tree, server)
	if err != nil {
		fmt.Println("Erro ao enviar a árvore:", err)
		return fmt.Errorf("erro ao enviar a árvore: %w", err)
	}

	fmt.Println("Árvore de versionamento atualizada, gerando árvore local...")
	tree, err := buildTree(path, &vCurrent.ExtensionsToGenerateVersion)
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

func PushRepository(path string, server string) error {
	if !VerifyIfExistVersionControl(path) {
		return fmt.Errorf("controle de versão não inicializado")
	}

	fmt.Println("Enviando HEAD para o servidor...")
	vCurrent, err := decompressVersionFile(path)

	if err != nil {
		fmt.Println("Erro ao ler a árvore salva:", err)
		return fmt.Errorf("erro ao ler a árvore salva: %w", err)
	}

	hasModifications := sendHeadOfVersion(vCurrent.Head, server)
	if !hasModifications {
		fmt.Println("Repositório já está atualizado.")
		return nil
	}

	fmt.Println("Repositório atualizado, enviando árvore de versionamento...")
	c, err := sendTreeOfVersion(&vCurrent.Tree, server)

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

	err = sendFilesToServer(buf, server)

	if err != nil {
		fmt.Println("Erro ao enviar os arquivos:", err)
		return fmt.Errorf("erro ao enviar os arquivos: %w", err)
	}

	fmt.Println("Arquivos enviados com sucesso.")
	return nil
}
