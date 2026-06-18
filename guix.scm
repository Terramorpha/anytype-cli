;;; guix.scm — package definition for anytype-cli.
;;;
;;; Build a GC-tracked binary from this checkout:
;;;
;;;   guix build -f guix.scm        ; result symlink is a GC root
;;;   guix shell -f guix.scm        ; ephemeral dev env
;;;   guix package -f guix.scm      ; install into a profile (survives `guix gc`)
;;;
;;; Notes:
;;; - The Go dependencies are vendored (run `go mod vendor` first; the vendor/
;;;   tree must be present in this checkout since the build runs offline).
;;; - tantivy is a prebuilt static lib fetched as a hash-pinned origin; it is a
;;;   Rust/C archive that `go mod vendor` cannot capture.
;;; - Built with `-tags noheic` to drop the libde265 C dependency (matches the
;;;   upstream Linux release build), which also is not vendorable.

(use-modules (guix packages)
             (guix gexp)
             (guix download)
             (guix build-system gnu)
             ((guix licenses) #:prefix license:)
             (gnu packages golang)
             (gnu packages commencement)
             (srfi srfi-1)
             (srfi srfi-13))

(define %source-dir (canonicalize-path (dirname (current-filename))))

(define tantivy-lib
  (origin
    (method url-fetch)
    (uri "https://github.com/anyproto/tantivy-go/releases/download/v1.0.6/linux-amd64-musl.tar.gz")
    (sha256
     (base32 "1g5bma200vc5v9ja2kpx0rfrgfvwyq2n9cp854nyrgq7cw04xcw8"))))

;; Keep the working tree but drop heavy/irrelevant top-level dirs. vendor/ is
;; kept (it is not git-tracked, so a git predicate would wrongly exclude it).
(define %excluded
  (map (lambda (d) (string-append %source-dir "/" d))
       '(".git" "dist" "result" ".gocache" ".gopath" ".direnv")))

(define (keep-source? file stat)
  (not (any (lambda (ex)
              (or (string=? file ex)
                  (string-prefix? (string-append ex "/") file)))
            %excluded)))

(package
  (name "anytype-cli")
  (version "0.0.0-6671438")
  (source (local-file %source-dir "anytype-cli-checkout"
                      #:recursive? #t
                      #:select? keep-source?))
  (build-system gnu-build-system)
  (arguments
   (list
    #:tests? #f
    #:phases
    #~(modify-phases %standard-phases
        (delete 'configure)
        (replace 'build
          (lambda _
            (let ((tantivy (string-append (getcwd) "/.tantivy")))
              (mkdir-p tantivy)
              (invoke "tar" "xzf" #$tantivy-lib "-C" tantivy)
              ;; Offline, vendored build environment.
              (setenv "HOME" (getcwd))
              (setenv "GOCACHE" (string-append (getcwd) "/.gocache"))
              (setenv "GOPATH" (string-append (getcwd) "/.gopath"))
              (setenv "GOFLAGS" "-mod=vendor")
              (setenv "GOTOOLCHAIN" "local")
              (setenv "GOPROXY" "off")
              (setenv "GOSUMDB" "off")
              (setenv "CGO_ENABLED" "1")
              (setenv "CGO_LDFLAGS" (string-append "-L" tantivy))
              (invoke "go" "build"
                      "-tags" "noheic"
                      ;; Strip build paths so the binary doesn't retain the Go
                      ;; toolchain / gcc-toolchain in its closure.
                      "-trimpath"
                      "-ldflags"
                      (string-append "-s -w -X "
                                     "'github.com/anyproto/anytype-cli/core.Version="
                                     #$version "'")
                      "-o" "anytype"))))
        (replace 'install
          (lambda _
            (let ((bin (string-append #$output "/bin")))
              (mkdir-p bin)
              (install-file "anytype" bin)
              ;; Mirror `make install`: `any` as a short alias.
              (symlink "anytype" (string-append bin "/any"))))))))
  (native-inputs (list go gcc-toolchain))
  (home-page "https://github.com/anyproto/anytype-cli")
  (synopsis "Command-line interface for Anytype with an embedded server")
  (description
   "anytype-cli is a self-contained Go command-line interface for Anytype.  It
embeds the anytype-heart middleware server, providing both server and client
functionality from a single binary, including space, chat, and profile
management over the local gRPC and HTTP APIs.")
  (license license:expat))
