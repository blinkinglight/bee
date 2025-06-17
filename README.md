# bee

requires "https://buf.build/docs/" and `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`


examples:

`go run ./examples/subscribers`

and

`go run ./examples/publishers`


tiny example of live projectios: 

```go 

	router.Get("/stream/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		id := chi.URLParam(r, "id")
		sse := datastar.NewSSE(w, r)
		_ = sse

		ctx := bee.WithJetStream(r.Context(), js)
		ctx = bee.WithNats(ctx, nc)

		agg := &Aggregate{}
		updates := bee.ReplayAndSubscribe(ctx, agg, ro.WithAggreate(users.Aggregate), ro.WithAggregateID(id))
		go func() {
			for {
				select {
				case <-r.Context().Done():
					return
				case update := <-updates:
					sse.MergeFragmentTempl(partials.History(update.History))
				}
			}
		}()
		<-r.Context().Done()
	})
```