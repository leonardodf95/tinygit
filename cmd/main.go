package main

import (
	"fmt"

	"github.com/leonardodf95/tinygit/internal/tinygit"
	"github.com/spf13/cobra"
)

var (
	path string
)

func main() {

	rootCmd := cobra.Command{}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(Init(), Status(), Commit())
	rootCmd.Execute()
}

func Init() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Inicializa o controle de versão",
		Run: func(cmd *cobra.Command, args []string) {
			err := tinygit.InitControlVersion(path)
			if err != nil {
				fmt.Println("Erro ao inicializar controle de versão:", err)
			}
		},
	}

	cmd.Flags().StringVarP(&path, "directory", "d", "", "Diretório de trabalho")

	return cmd
}

func Status() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Mostra o status de alterações do diretório monitorado pelo controle de versão",
		Run: func(cmd *cobra.Command, args []string) {
			_, err := tinygit.StatusControlVersion(path)
			if err != nil {
				fmt.Println("Erro ao verificar status:", err)
			}
		},
	}

	cmd.Flags().StringVarP(&path, "directory", "d", "", "Diretório de trabalho")

	return cmd
}

func Commit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit",
		Short: "Salva as mudanças no controle de versão",
		Run: func(cmd *cobra.Command, args []string) {
			err := tinygit.CommitControlVersion(path)
			if err != nil {
				fmt.Println("Erro ao realizar commit:", err)
			}
		},
	}

	cmd.Flags().StringVarP(&path, "directory", "d", "", "Diretório de trabalho")

	return cmd
}
