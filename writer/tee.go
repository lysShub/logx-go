package writer

type tee struct{ ws []Writer }

var _ Writer = (*tee)(nil)

func Tee(ws ...Writer) Writer { return &tee{ws: ws} }

func (t *tee) Write(p []byte) (n int, err error) {

	for _, w := range t.ws {
		_, e := w.Write(p)
		if e != nil && err == nil {
			err = e
		}
	}
	return len(p), err
}

func (t *tee) Sync() error {
	var err error
	for _, w := range t.ws {
		if e := w.Sync(); e != nil && err == nil {
			err = e
		}
	}
	return err
}

func (t *tee) Close() error {
	var err error
	for _, w := range t.ws {
		if e := w.Close(); e != nil && err == nil {
			err = e
		}
	}
	return err
}
