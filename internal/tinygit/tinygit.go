package tinygit

import "fmt"

// Estrutura para representar um nó na árvore
type Node struct {
	Path     string  `json:"path"`
	Hash     string  `json:"hash"`
	Type     string  `json:"type"` // "blob" para arquivos, "tree" para diretórios
	Children []*Node `json:"children,omitempty"`
}

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

var acceptedExtensions = []string{".exe", ".map", ".fr3", ".dll", ".xsd", ".wav", ".jpg"}

func InitControlVersion(path string) error {
	err := generateVersionDir(path)
	if err != nil {
		fmt.Println("Erro ao criar diretório de versão:", err)
		return err
	}
	fmt.Println("Controle de versão inicializado em", path)
	tree, err := buildTree(path)
	if err != nil {
		fmt.Println("Erro ao construir a árvore:", err)
		return err
	}
	if tree == nil {
		fmt.Println("A árvore está vazia.")
		return nil
	}

	err = generateVersionFile(path, *tree)
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
	err = generateVersionFile(path, *c.Modified[0])
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
	currentTree, err := buildTree(path)
	if err != nil {
		fmt.Println("Erro ao construir a árvore:", err)
		return nil, err
	}
	savedTree, err := decompressVersionFile(path)
	if err != nil {
		fmt.Println("Erro ao ler a árvore salva:", err)
		return nil, err
	}
	changes := compareTrees(savedTree, currentTree)

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
