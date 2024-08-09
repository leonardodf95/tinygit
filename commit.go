package main

import "fmt"

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
