package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/leonardodf95/tinygit/internal/tinygit"
)

func main() {

	// Definindo as flags
	directoryFlag := flag.String("d", "", "Diretório de trabalho")

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
		path = filepath.Clean(*directoryFlag)
	} else {

		pathAbs, err := filepath.Abs(".")
		if err != nil {
			fmt.Println("Erro ao determinar o diretório de trabalho:", err)
			return
		}
		path = pathAbs
	}

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
		err := tinygit.InitControlVersion(path)
		if err != nil {
			fmt.Println("Erro ao inicializar controle de versão:", err)
		}
	case "status":
		_, err := tinygit.StatusControlVersion(path)
		if err != nil {
			fmt.Println("Erro ao verificar status:", err)
		}
	case "commit":
		err := tinygit.CommitControlVersion(path)
		if err != nil {
			fmt.Println("Erro ao realizar commit:", err)
		}
	default:
		fmt.Println("Comando desconhecido:", command)
	}
}
