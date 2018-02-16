package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
)

const (
	ipPrefix  = "IPS-"
	txtPrefix = "TXT-"
	crtPrefix = "CERT-"
)

type FileDatabase string

func (db FileDatabase) getFile(ctx context.Context, name string) ([]byte, error) {
	name = filepath.Join(string(db), name)
	var (
		data []byte
		err  error
		done = make(chan struct{})
	)
	go func() {
		data, err = ioutil.ReadFile(name)
		close(done)
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
	}
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

func (db FileDatabase) putFile(ctx context.Context, name string, data []byte) error {
	if err := os.MkdirAll(string(db), 0700); err != nil {
		return err
	}

	done := make(chan struct{})
	var err error
	go func() {
		defer close(done)
		var tmp string
		if tmp, err = db.writeTempFile(name, data); err != nil {
			return
		}
		select {
		case <-ctx.Done():
			// Don't overwrite the file if the context was canceled.
		default:
			newName := filepath.Join(string(db), name)
			err = os.Rename(tmp, newName)
		}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}
	return err
}

func (db FileDatabase) deleteFile(ctx context.Context, name string) error {
	name = filepath.Join(string(db), name)
	var (
		err  error
		done = make(chan struct{})
	)
	go func() {
		err = os.Remove(name)
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// writeTempFile writes b to a temporary file, closes the file and returns its path.
func (db FileDatabase) writeTempFile(prefix string, b []byte) (string, error) {
	// TempFile uses 0600 permissions
	f, err := ioutil.TempFile(string(db), prefix)
	if err != nil {
		return "", err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return "", err
	}
	return f.Name(), f.Close()
}

func encodeToGOB(m interface{}) ([]byte, error) {
	var buf bytes.Buffer

	e := gob.NewEncoder(&buf)
	if err := e.Encode(m); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func decodeFromGOB(b []byte, m interface{}) error {
	var buf bytes.Buffer

	buf.Write(b)
	d := gob.NewDecoder(&buf)
	if err := d.Decode(m); err != nil {
		return err
	}
	return nil
}

func (db FileDatabase) DoesDomainExist(ctx context.Context, domain string) (bool, error) {
	var (
		hasIP  bool
		hasTXT bool
	)

	ipaddrs, err := db.GetIPAddresses(ctx, domain)
	if err == nil {
		hasIP = len(ipaddrs) > 0
	} else {
		return false, err
	}

	txtvals, err := db.GetTXTValues(ctx, domain)
	if err == nil {
		hasTXT = len(txtvals) > 0
	} else {
		return false, err
	}
	return hasIP || hasTXT, nil
}

func (db FileDatabase) GetIPAddresses(ctx context.Context, domain string) ([]net.IP, error) {
	var addresses []net.IP

	bytes, err := db.getFile(ctx, ipPrefix+domain)
	if bytes == nil {
		return nil, err
	}
	if err := decodeFromGOB(bytes, &addresses); err != nil {
		return nil, err
	}

	return addresses, nil
}

func (db FileDatabase) PutIPAddresses(ctx context.Context, domain string, addresses []net.IP) error {
	bytes, err := encodeToGOB(addresses)
	if err != nil {
		return err
	}
	return db.putFile(ctx, ipPrefix+domain, bytes)
}

func (db FileDatabase) DeleteIPAddresses(ctx context.Context, domain string) error {
	return db.deleteFile(ctx, ipPrefix+domain)
}

func (db FileDatabase) GetTXTValues(ctx context.Context, domain string) ([]string, error) {
	var values []string

	bytes, err := db.getFile(ctx, txtPrefix+domain)
	if bytes == nil {
		return nil, err
	}
	if err := decodeFromGOB(bytes, &values); err != nil {
		return nil, err
	}

	return values, nil
}

func (db FileDatabase) PutTXTValues(ctx context.Context, domain string, values []string) error {
	bytes, err := encodeToGOB(values)
	if err != nil {
		return err
	}
	return db.putFile(ctx, txtPrefix+domain, bytes)
}

func (db FileDatabase) DeleteTXTValues(ctx context.Context, domain string) error {
	return db.deleteFile(ctx, txtPrefix+domain)
}

func (db FileDatabase) GetCertificate(ctx context.Context, domain string) ([]byte, error) {
	return db.getFile(ctx, crtPrefix+domain)
}

func (db FileDatabase) PutCertificate(ctx context.Context, domain string, data []byte) error {
	return db.putFile(ctx, crtPrefix+domain, data)
}

func (db FileDatabase) DeleteCertificate(ctx context.Context, domain string) error {
	return db.deleteFile(ctx, crtPrefix+domain)
}
