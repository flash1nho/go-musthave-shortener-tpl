//go:build ignore

package handler

func (r *ShortenRequest) Reset() {
	r.URL = ""
}
func (r *ShortenResponse) Reset() {
	r.Result = ""
}
func (r *BatchShortenRequest) Reset() {
	r.CorrelationID = ""
	r.OriginalURL = ""
}
func (r *BatchShortenResponse) Reset() {
	r.CorrelationID = ""
	r.ShortURL = ""
}
func (r *BatchUserShortenResponse) Reset() {
	r.ShortURL = ""
	r.OriginalURL = ""
}
func (r *Batch) Reset() {
	clear(r.urlMappings)
}
func (r *Handler) Reset() {
	if r.store != nil {
		*r.store.Reset()
	}
	r.server.Reset()
	if r.log != nil {
		*r.log.Reset()
	}
}
