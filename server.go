package tinygit

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func CompareHeadsHandler(w http.ResponseWriter, r *http.Request, head string) {
	rHead := r.URL.Query().Get("head")

	if rHead == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hasChanges := CompareHashes(head, rHead)

	if hasChanges {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotModified)
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

func PullHandler(w http.ResponseWriter, r *http.Request, n Node) {
	ctx := r.Context()
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

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	jsonWriter, err := writer.CreateFormField("changes")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	jsonBytes, err := json.Marshal(c)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = jsonWriter.Write(jsonBytes)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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

			err = filepath.Walk(n.Path, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				for _, node := range c.Modified {
					if node.Path != path {
						continue
					}
					relPath := strings.TrimPrefix(path, n.Path)
					relPath = strings.TrimPrefix(relPath, string(filepath.Separator))

					err = addFileToZip(zipWriter, path, relPath, info)
					if err != nil {
						return err
					}
				}

				for _, node := range c.Added {
					if node.Path != path {
						continue
					}
					relPath := strings.TrimPrefix(path, n.Path)
					relPath = strings.TrimPrefix(relPath, string(filepath.Separator))

					err = addFileToZip(zipWriter, path, relPath, info)
					if err != nil {
						return err
					}
				}

				return nil
			})

		}()

		w.Header().Set("Content-Type", writer.FormDataContentType())
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
