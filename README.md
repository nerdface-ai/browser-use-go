# browser-use-go

A [browser-use](https://github.com/browser-use/browser-use) implementation in Go.

### Install the browsers and OS dependencies
```
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps
# Or
go install github.com/playwright-community/playwright-go/cmd/playwright@latest
playwright install --with-deps
```

### Execute
```
go run ./browser-use/cmd
```


# Plan

- [x] follow directory structure [link](https://5takoo.tistory.com/378)
- [] copy `dom` from browser-use
    - [x] clickable_element_processor.go
    - [x] history_tree_processor.go
    - [x] views.go
    - [] service.go
- [] copy `controller` from browser-use
- [] copy `browser` from browser-use
- [] google search keyword 'browser-use'