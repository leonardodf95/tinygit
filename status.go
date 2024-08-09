package main

import "fmt"

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
