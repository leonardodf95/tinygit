package tinygit

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func RequestClone(url string, path string) (*Versioning, error) {
	// Verificar se o diretório já existe
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, errors.New("diretório não existe")
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Println("Status:", resp.Status)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("erro ao baixar o repositório")
	}

	tempFile, err := os.CreateTemp("", "tinygit")
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

func sendHeadOfVersion(head string, url string) bool {
	req, err := http.NewRequest(http.MethodGet, url+"/head", nil)
	if err != nil {
		fmt.Println("Erro ao criar requisição:", err)
		return false
	}

	q := req.URL.Query()
	q.Add("head", head)
	req.URL.RawQuery = q.Encode()

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

func sendTreeOfVersionForUpdate(tree *Node, url string) error {
	req, err := http.NewRequest(http.MethodPost, url+"/pull", nil)
	if err != nil {
		return err
	}

	b, err := json.Marshal(tree)
	if err != nil {
		return err
	}

	req.Body = io.NopCloser(strings.NewReader(string(b)))

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.New("erro ao enviar árvore")
	}

	// GERANDO ARQUIVOS TEMPORÁRIOS
	timeStamp := time.Now().Format("20060102150405")
	tempFile, err := os.CreateTemp("", "tinygit"+timeStamp)

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

	err = unzipFiles(tempFile.Name(), tree.Path)
	if err != nil {
		return err
	}

	for _, removed := range resp.Header["Removed"] {
		err = os.RemoveAll(filepath.Join(tree.Path, removed))
		if err != nil {
			return err
		}
	}

	return nil
}

func sendTreeOfVersion(tree *Node, url string) (*Changes, error) {
	req, err := http.NewRequest(http.MethodPost, url+"/tree", nil)
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
func sendFilesToServer(b []bytes.Buffer, url string) error {
	req, err := http.NewRequest(http.MethodPost, url+"/push", &b[0])
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
