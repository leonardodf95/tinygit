package main

import (
	"flag"
	"fmt"
	"os"
)

var acceptedExtensions = []string{".exe", ".map", ".fr3", ".dll", ".xsd", ".wav", ".jpg"}

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

func main() {

	// Definindo as flags
	directoryFlag := flag.String("d", "", "Diretório de trabalho")
	directoryLongFlag := flag.String("directory", "", "Diretório de trabalho")

	fmt.Println("diretoryFlag:", *directoryFlag)
	fmt.Println("directoryLongFlag:", *directoryLongFlag)

	// Fazendo o parse das flags
	flag.Parse()

	// Verificando se há argumentos suficientes
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run main.go <comando> [opções]")
		return
	}
	command := os.Args[1]

	// Determina o diretório usando a flag ou o argumento padrão
	var path string
	if *directoryFlag != "" {
		path = *directoryFlag
	} else if *directoryLongFlag != "" {
		path = *directoryLongFlag
	} else {
		path = "."
	}

	fmt.Println("Comando:", command)

	fmt.Println("Diretório de trabalho:", path)

	switch command {
	case "help":
		fmt.Println("Comandos disponíveis:")
		fmt.Println("	init - Inicializa o controle de versão")
		fmt.Println("		Uso: init -d / --directory <diretório>")
		fmt.Println("	status - Mostra o status das mudanças")
		fmt.Println("		Uso: status -d / --directory <diretório>")
		fmt.Println("	commit - Salva as mudanças no controle de versão")
		fmt.Println("		Uso: commit -d / --directory <diretório>")
		fmt.Println("	help - Mostra esta mensagem de ajuda")
	case "init":
		err := InitControlVersion(path)
		if err != nil {
			fmt.Println("Erro ao inicializar controle de versão:", err)
		}
	case "status":
		_, err := StatusControlVersion(path)
		if err != nil {
			fmt.Println("Erro ao verificar status:", err)
		}
	case "commit":
		err := CommitControlVersion(path)
		if err != nil {
			fmt.Println("Erro ao realizar commit:", err)
		}
	default:
		fmt.Println("Comando desconhecido:", command)
	}
}
