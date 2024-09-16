package tinygit

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func RequestClone(path string, serverUrl string, parameters map[string]string) (*Versioning, error) {
	// Verificar se o diretório já existe
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, errors.New("diretório não existe")
	}

	u, err := parseUrlParameter(serverUrl, "clone", parameters)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.Status)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("erro ao baixar o repositório")
	}

	id := uuid.New().String()
	tempFile, err := os.CreateTemp("", id+".zip")
	if err != nil {
		return nil, err
	}

	defer os.Remove(tempFile.Name())

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return nil, err
	}

	err = tempFile.Close()

	if err != nil {
		return nil, err
	}

	err = unzipFiles(tempFile.Name(), path)
	if err != nil {
		return nil, err
	}

	ext := resp.Header.Get("Config-Ext")

	if ext == "" {
		return nil, errors.New("extensão de configuração não encontrada")
	}

	v := Versioning{
		ExtensionsToGenerateVersion: strings.Split(ext, ","),
	}

	return &v, nil
}

func sendHeadOfVersion(head string, serverUrl string, parameters map[string]string) bool {
	parameters["head"] = head
	u, err := parseUrlParameter(serverUrl, "head", parameters)
	if err != nil {
		fmt.Println("Erro ao criar URL:", err)
		return false
	}

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		fmt.Println("Erro ao criar requisição:", err)
		return false
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Erro ao enviar HEAD:", err)
		return false
	}

	if resp.StatusCode == http.StatusNotModified {
		return false
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Erro ao enviar HEAD:", resp.Status)
		return false
	}

	return true
}

func sendTreeOfVersionForUpdate(rootPath string, tree *Node, serverUrl string, parameters map[string]string) error {
	u, err := parseUrlParameter(serverUrl, "pull", parameters)
	if err != nil {
		fmt.Println("aqui 3", err)
		return err
	}

	fmt.Println("Enviando árvore para o servidor...")

	b, err := json.Marshal(tree)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(b))
	if err != nil {
		fmt.Println("aqui 1", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("aqui 2", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("erro ao enviar árvore, status: " + resp.Status)
	}
	fmt.Println("Árvore enviada com sucesso! Processando resposta...")
	//CRIAR ARQUIVO TEMPORÁRIO
	id := uuid.New().String()
	tempFile, err := os.CreateTemp("", id+".zip")
	if err != nil {
		return err
	}

	defer os.Remove(tempFile.Name())

	_, err = io.Copy(tempFile, resp.Body)

	if err != nil {
		return err
	}

	err = tempFile.Close()

	if err != nil {
		return err
	}

	err = unzipFiles(tempFile.Name(), filepath.Join(rootPath))
	if err != nil {
		return err
	}

	removedRaw := resp.Header.Get("Removed")
	var removed []string
	if removedRaw != "" {
		removed = strings.Split(removedRaw, ",")
	}

	if len(removed) > 0 {
		fmt.Println("Removendo:")
		for _, r := range removed {
			fmt.Println(r)
			err := os.RemoveAll(filepath.Join(rootPath, r))
			if err != nil {
				fmt.Println("Erro ao remover o arquivo:", err)
			}
		}
	}

	return nil
}

func sendTreeOfVersion(tree *Node, serverUrl string, paramenters map[string]string) (*Changes, error) {
	u, err := parseUrlParameter(serverUrl, "tree", paramenters)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, u, nil)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(tree)
	if err != nil {
		return nil, err
	}

	req.Body = io.NopCloser(strings.NewReader(string(b)))

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("erro ao enviar árvore")
	}

	c := Changes{
		Added:    []*Node{},
		Removed:  []*Node{},
		Modified: []*Node{},
	}

	err = json.NewDecoder(resp.Body).Decode(&c)

	if err != nil {
		return nil, err
	}

	if len(c.Modified) > 0 {
		m := []Node{}
		for _, modified := range c.Modified {
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

	if len(c.Added) > 0 {
		a := []Node{}
		for _, added := range c.Added {
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

	if len(c.Removed) > 0 {
		r := []Node{}
		for _, removed := range c.Removed {
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

	return &c, nil
}

// sendFilesToServer envia o arquivo compactado para o servidor
func sendFilesToServer(b []bytes.Buffer, serverUrl string, parameters map[string]string) error {
	u, err := parseUrlParameter(serverUrl, "push", parameters)

	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, u, &b[0])
	if err != nil {
		return err
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("erro ao enviar arquivos")
	}

	return nil
}

func parseUrlParameter(serverUrl, path string, parameters map[string]string) (string, error) {
	u, err := url.Parse(serverUrl)
	if err != nil {
		return "", err
	}
	u.Path = filepath.Join(u.Path, path)
	q := u.Query()
	for k, v := range parameters {
		q.Add(k, v)
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func unzipFiles(zipPath string, destPath string) error {
	r, err := zip.OpenReader(zipPath)

	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destPath, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(destPath)+string(os.PathSeparator)) {
			return fmt.Errorf("%s: invalid file path", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		destFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		fileInArchive, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(destFile, fileInArchive)

		if err != nil {
			return err
		}

		err = destFile.Close()

		if err != nil {
			return err
		}

		err = fileInArchive.Close()

		if err != nil {
			return err
		}

		err = os.Chtimes(fpath, f.Modified, f.Modified)
		if err != nil {
			return err
		}

	}

	return nil
}
