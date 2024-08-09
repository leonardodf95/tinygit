package main

import "fmt"

func InitControlVersion(path string) error {
	err := generateVersionDir(path)
	if err != nil {
		fmt.Println("Erro ao criar diretório de versão:", err)
		return err
	}
	tree, err := buildTree(path)
	if err != nil {
		fmt.Println("Erro ao construir a árvore:", err)
		return err
	}
	err = generateVersionFile(path, *tree)
	if err != nil {
		fmt.Println("Erro ao salvar a árvore:", err)
		return err
	}
	return nil
}
