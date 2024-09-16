package tinygit

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func CompareHeadsHandler(w http.ResponseWriter, r *http.Request, path string) {
	rHead := r.URL.Query().Get("head")

	fmt.Println("RHEAD:", rHead)
	if rHead == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tree, err := GetTreeControlVersion(path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Println("HEAD:", tree.Hash)

	hasNoChanges := CompareHashes(tree.Hash, rHead)

	if hasNoChanges {
		w.WriteHeader(http.StatusNotModified)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func CompareTreesHandler(w http.ResponseWriter, r *http.Request, n Node) {
	//Ler Arvore do corpo da requisição
	rawTree, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var tree Node
	err = json.Unmarshal(rawTree, &tree)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//Comparar arvores
	c := CompareTrees(&n, &tree)

	//Retornar alterações
	b, err := json.Marshal(c)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(b)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
}

func PullHandler(w http.ResponseWriter, r *http.Request, rootPath string, n Node) {

	fmt.Println("n.Path:", n.Path)
	fmt.Println("n.Hash:", n.Hash)
	fmt.Println("n.Type:", n.Type)

	ctx := r.Context()
	//Ler Arvore do corpo da requisição
	rawTree, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("ERRO AO LER O CORPO DA REQUISIÇÃO:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var tree Node
	err = json.Unmarshal(rawTree, &tree)
	if err != nil {
		fmt.Println("ERRO AO DECODIFICAR JSON:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//Comparar arvores
	c := CompareTrees(&tree, &n)
	for _, node := range c.Removed {
		fmt.Println("REMOVED NODE PATH:", node.Path)
	}
	for _, node := range c.Modified {
		fmt.Println("MODIFIED NODE PATH:", node.Path)
	}
	for _, node := range c.Added {
		fmt.Println("ADDED NODE PATH:", node.Path)
	}

	pr, pw := io.Pipe()
	zipWriter := zip.NewWriter(pw)
	removed := []string{}
	for _, node := range c.Removed {
		fmt.Println("REMOVED NODE PATH:", node.Path)
		removed = append(removed, node.Path)
	}

	select {
	case <-ctx.Done():
		// Operação foi cancelada
		fmt.Println("REQUISIÇÃO CANCELADA")
		http.Error(w, "Request canceled", http.StatusRequestTimeout)
		return
	default:

		go func() {
			defer pw.Close()
			defer zipWriter.Close()

			fmt.Println("n PATH:", n.Path)
			fmt.Println("ROOT PATH:", rootPath)

			err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				relPath := strings.TrimPrefix(path, rootPath)
				relPath = strings.TrimPrefix(relPath, string(filepath.Separator))

				for _, node := range c.Modified {
					if node.Path != relPath {
						continue
					}

					fmt.Println("ARQUIVO MODIFICADO:", relPath)
					err = addFileToZip(zipWriter, path, relPath, info)
					if err != nil {
						return err
					}
				}

				for _, node := range c.Added {
					logFile, err := os.OpenFile(filepath.Join(rootPath, "log.txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						return err
					}
					defer logFile.Close()
					logFile.WriteString(fmt.Sprintf("path: %s", path))
					logFile.WriteString(fmt.Sprintf("node path: %s\n", node.Path))
					if node.Path != relPath {
						continue
					}

					fmt.Println("ARQUIVO ADICIONADO", relPath)
					err = addFileToZip(zipWriter, path, relPath, info)
					if err != nil {
						return err
					}
				}

				return nil
			})
			if err != nil {
				fmt.Println("ERRO AO PERCORRER DIRETÓRIO:", err)
			}

		}()

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=pull.zip")
		w.Header().Set("Removed", strings.Join(removed, ","))
		w.WriteHeader(http.StatusOK)
		io.Copy(w, pr)
	}
}

// PushFilesHandler Recebe arquivos e atualiza a arvore
func PushFilesHandler(w http.ResponseWriter, r *http.Request, rootPath string) {
	// Verifica se o método é POST
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Lê o arquivo zip do corpo da requisição
	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, r.Body)
	if err != nil {
		http.Error(w, "Erro ao ler o corpo da requisição", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Salva o arquivo zip recebido em disco
	tempZipFile, err := os.CreateTemp("", "tinygit-"+time.Now().Format("20060102150405")+".zip")
	if err != nil {
		http.Error(w, "Erro ao criar arquivo temporário", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempZipFile.Name())

	_, err = tempZipFile.Write(buf.Bytes())
	if err != nil {
		http.Error(w, "Erro ao salvar arquivo temporário", http.StatusInternalServerError)
		return
	}

	// Descompacta o arquivo zip
	err = unzipFiles(tempZipFile.Name(), rootPath)

	if err != nil {
		http.Error(w, "Erro ao descompactar arquivos", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func CloneHandler(w http.ResponseWriter, r *http.Request, rootPath string) {
	ctx := r.Context()
	// Verifica se o método é GET
	if r.Method != http.MethodGet {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Verifica se o diretório já existe
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		http.Error(w, "Diretório não existe", http.StatusNotFound)
		return
	}

	v, err := decompressVersionFile(rootPath)
	if err != nil {
		http.Error(w, "Erro ao ler a árvore salva", http.StatusInternalServerError)
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
		pr, pw := io.Pipe()
		zipWriter := zip.NewWriter(pw)

		go func() {
			defer pw.Close()
			defer zipWriter.Close()

			err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				ext := filepath.Ext(path)
				if !contains(v.ExtensionsToGenerateVersion, ext) {
					return nil
				}

				relPath := strings.TrimPrefix(path, rootPath)
				relPath = strings.TrimPrefix(relPath, string(filepath.Separator))

				err = addFileToZip(zipWriter, path, relPath, info)
				if err != nil {

					return err
				}
				return nil
			})

			if err != nil {
				http.Error(w, "Erro ao percorrer diretório", http.StatusInternalServerError)
				return

			}
		}()

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=clone.zip")
		w.Header().Set("Config-Ext", strings.Join(v.ExtensionsToGenerateVersion, ","))
		w.WriteHeader(http.StatusOK)
		io.Copy(w, pr)
	}
}
