# « Heures de bénévolat » — compilation et paquets de distribution.
#
#   make              liste les cibles
#   make run          lance l'application
#   make check        gofmt + go vet + tests
#   make dist         construit tous les paquets dans dist/
#   make install      installe l'application sur cette machine
#
# macOS passe par Cocoa (cgo) : les paquets macOS se construisent donc sur un
# Mac. Windows et Linux se compilent sans cgo, depuis n'importe quelle machine.

APP      := HB_version5
BIN      := hb_go_version5
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GO       ?= go

# Les deux modes d'emploi ne servent pas sur la même plateforme : chaque
# paquet n'embarque que le sien (7 Mo de docx en moins par archive).
MANUAL_MACOS   := Mode_d_emploi_benevolat_MAC_v5.docx
MANUAL_DESKTOP := Mode_d_emploi_benevolat_v5.docx

BUILD_DIR := build
DIST_DIR  := dist
STAGE_DIR := $(BUILD_DIR)/stage
DIST_ABS  := $(abspath $(DIST_DIR))
LDFLAGS   := -X main.version=$(VERSION)
GO_FILES  := $(wildcard *.go)
HOST_OS   := $(shell $(GO) env GOOS)

# Installation locale (make install) : ~/.local sous Linux, ~/Applications
# sous macOS. Redéfinissable : make install PREFIX=/opt/hb
PREFIX    ?= $(HOME)/.local
MACOS_APPS ?= $(HOME)/Applications

.DEFAULT_GOAL := help

.PHONY: help run build test vet fmt check screens clean distclean dist \
	dist-macos dist-macos-intel dist-macos-silicon dist-macos-universal \
	dist-windows dist-windows-amd64 dist-windows-arm64 \
	dist-linux dist-linux-amd64 dist-linux-arm64 \
	checksums install uninstall version

help: ## Affiche cette aide
	@echo "$(APP) $(VERSION)"
	@echo
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk -F':.*?## ' '{printf "  %-24s %s\n", $$1, $$2}'
	@echo
	@echo "  Paquets produits dans $(DIST_DIR)/ :"
	@echo "    $(APP)-$(VERSION)-macos-{intel,apple-silicon,universal}.zip"
	@echo "    $(APP)-$(VERSION)-windows-{amd64,arm64}.zip"
	@echo "    $(APP)-$(VERSION)-linux-{amd64,arm64}.tar.gz"

version: ## Affiche la version qui sera compilée
	@echo $(VERSION)

# ----------------------------------------------------------- développement

run: ## Lance l'application
	$(GO) run -ldflags "$(LDFLAGS)" .

build: $(BUILD_DIR)/$(BIN) ## Compile pour cette machine (build/)

$(BUILD_DIR)/$(BIN): $(GO_FILES) go.mod go.sum
	@mkdir -p $(BUILD_DIR)
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $@ .

test: ## Exécute les tests
	$(GO) test ./...

vet: ## Analyse statique
	$(GO) vet ./...

fmt: ## Reformate le code
	gofmt -w .

check: ## Vérifie le format, l'analyse statique et les tests
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "gofmt : fichiers à reformater :"; gofmt -l .; exit 1; \
	fi
	$(GO) vet ./...
	$(GO) test ./...

screens: build ## Rend tous les écrans en PNG (build/screens/)
	@mkdir -p $(BUILD_DIR)/screens
	@for s in $$($(BUILD_DIR)/$(BIN) -screens); do \
		echo "  $$s"; \
		$(BUILD_DIR)/$(BIN) -png $(BUILD_DIR)/screens/$$s.png -screen $$s || exit 1; \
	done

clean: ## Supprime les fichiers de compilation
	rm -rf $(BUILD_DIR)

distclean: clean ## Supprime aussi les paquets
	rm -rf $(DIST_DIR)

# ------------------------------------------------------------------ macOS
#
# Un vrai bundle .app est assemblé à la main (Info.plist, icône, ressources)
# puis signé « ad hoc » : sans signature, macOS refuse de lancer un binaire
# arm64, et lipo invalide de toute façon la signature posée par le linker.

$(BUILD_DIR)/icon.icns: Resources/Image_logiciel.png
	@mkdir -p $(BUILD_DIR)/icon.iconset
	@sips -Z 1024 --padToHeightWidth 1024 1024 --padColor FFFFFF $< \
		--out $(BUILD_DIR)/icon-master.png >/dev/null
	@for s in 16 32 128 256 512; do \
		sips -z $$s $$s $(BUILD_DIR)/icon-master.png \
			--out $(BUILD_DIR)/icon.iconset/icon_$${s}x$${s}.png >/dev/null; \
		d=$$((s * 2)); \
		sips -z $$d $$d $(BUILD_DIR)/icon-master.png \
			--out $(BUILD_DIR)/icon.iconset/icon_$${s}x$${s}@2x.png >/dev/null; \
	done
	iconutil -c icns $(BUILD_DIR)/icon.iconset -o $@

# $(1) = GOARCH
define macos_binary
	@mkdir -p $(BUILD_DIR)/macos
	CGO_ENABLED=1 GOOS=darwin GOARCH=$(1) $(GO) build -trimpath \
		-ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/macos/$(BIN)-$(1) .
endef

# $(1) = étiquette du paquet, $(2) = binaire à embarquer
define macos_package
	rm -rf $(STAGE_DIR)/macos-$(1)
	mkdir -p $(STAGE_DIR)/macos-$(1)/$(APP).app/Contents/MacOS
	mkdir -p $(STAGE_DIR)/macos-$(1)/$(APP).app/Contents/Resources
	cp $(2) $(STAGE_DIR)/macos-$(1)/$(APP).app/Contents/MacOS/$(BIN)
	cp -R Resources/. $(STAGE_DIR)/macos-$(1)/$(APP).app/Contents/Resources/
	rm -f $(STAGE_DIR)/macos-$(1)/$(APP).app/Contents/Resources/$(MANUAL_DESKTOP)
	cp $(BUILD_DIR)/icon.icns $(STAGE_DIR)/macos-$(1)/$(APP).app/Contents/Resources/icon.icns
	sed -e 's/@VERSION@/$(VERSION)/g' -e 's/@BIN@/$(BIN)/g' packaging/Info.plist.in \
		> $(STAGE_DIR)/macos-$(1)/$(APP).app/Contents/Info.plist
	cp packaging/INSTALLATION-macos.txt $(STAGE_DIR)/macos-$(1)/INSTALLATION.txt
	xattr -cr $(STAGE_DIR)/macos-$(1)/$(APP).app
	codesign --force --sign - --timestamp=none $(STAGE_DIR)/macos-$(1)/$(APP).app
	@mkdir -p $(DIST_DIR)
	rm -f $(DIST_ABS)/$(APP)-$(VERSION)-macos-$(1).zip
	ditto -c -k --norsrc --noextattr $(STAGE_DIR)/macos-$(1) \
		$(DIST_ABS)/$(APP)-$(VERSION)-macos-$(1).zip
	@echo "  → $(DIST_DIR)/$(APP)-$(VERSION)-macos-$(1).zip"
endef

dist-macos: dist-macos-intel dist-macos-silicon dist-macos-universal ## Paquets macOS (Intel, Apple Silicon, universel)

dist-macos-intel: $(BUILD_DIR)/icon.icns ## Paquet macOS Intel (.zip)
	$(call macos_binary,amd64)
	$(call macos_package,intel,$(BUILD_DIR)/macos/$(BIN)-amd64)

dist-macos-silicon: $(BUILD_DIR)/icon.icns ## Paquet macOS Apple Silicon (.zip)
	$(call macos_binary,arm64)
	$(call macos_package,apple-silicon,$(BUILD_DIR)/macos/$(BIN)-arm64)

dist-macos-universal: $(BUILD_DIR)/icon.icns ## Paquet macOS universel Intel + Apple Silicon (.zip)
	$(call macos_binary,amd64)
	$(call macos_binary,arm64)
	lipo -create -output $(BUILD_DIR)/macos/$(BIN)-universal \
		$(BUILD_DIR)/macos/$(BIN)-amd64 $(BUILD_DIR)/macos/$(BIN)-arm64
	$(call macos_package,universal,$(BUILD_DIR)/macos/$(BIN)-universal)

# ---------------------------------------------------------------- Windows
#
# -H=windowsgui : application graphique, sans fenêtre de console derrière.

# $(1) = GOARCH
define windows_package
	rm -rf $(STAGE_DIR)/windows-$(1)
	mkdir -p $(STAGE_DIR)/windows-$(1)/$(APP)
	CGO_ENABLED=0 GOOS=windows GOARCH=$(1) $(GO) build -trimpath \
		-ldflags "-H=windowsgui $(LDFLAGS)" \
		-o $(STAGE_DIR)/windows-$(1)/$(APP)/$(BIN).exe .
	cp -R Resources $(STAGE_DIR)/windows-$(1)/$(APP)/Resources
	rm -f $(STAGE_DIR)/windows-$(1)/$(APP)/Resources/$(MANUAL_MACOS)
	cp packaging/INSTALLATION-windows.txt $(STAGE_DIR)/windows-$(1)/$(APP)/INSTALLATION.txt
	@mkdir -p $(DIST_DIR)
	rm -f $(DIST_ABS)/$(APP)-$(VERSION)-windows-$(1).zip
	cd $(STAGE_DIR)/windows-$(1) && zip -qr $(DIST_ABS)/$(APP)-$(VERSION)-windows-$(1).zip $(APP)
	@echo "  → $(DIST_DIR)/$(APP)-$(VERSION)-windows-$(1).zip"
endef

dist-windows: dist-windows-amd64 dist-windows-arm64 ## Paquets Windows (x86-64 et ARM64)

dist-windows-amd64: ## Paquet Windows x86-64 (.zip)
	$(call windows_package,amd64)

dist-windows-arm64: ## Paquet Windows ARM64 (.zip)
	$(call windows_package,arm64)

# ------------------------------------------------------------------ Linux

# $(1) = GOARCH
define linux_package
	rm -rf $(STAGE_DIR)/linux-$(1)
	mkdir -p $(STAGE_DIR)/linux-$(1)/$(APP)
	CGO_ENABLED=0 GOOS=linux GOARCH=$(1) $(GO) build -trimpath \
		-ldflags "$(LDFLAGS)" -o $(STAGE_DIR)/linux-$(1)/$(APP)/$(BIN) .
	cp -R Resources $(STAGE_DIR)/linux-$(1)/$(APP)/Resources
	rm -f $(STAGE_DIR)/linux-$(1)/$(APP)/Resources/$(MANUAL_MACOS)
	cp packaging/INSTALLATION-linux.txt $(STAGE_DIR)/linux-$(1)/$(APP)/INSTALLATION.txt
	install -m 0755 packaging/install-linux.sh $(STAGE_DIR)/linux-$(1)/$(APP)/install.sh
	sed 's/@VERSION@/$(VERSION)/g' packaging/hb_go_version5.desktop.in \
		> $(STAGE_DIR)/linux-$(1)/$(APP)/$(BIN).desktop
	@mkdir -p $(DIST_DIR)
	tar -czf $(DIST_ABS)/$(APP)-$(VERSION)-linux-$(1).tar.gz \
		-C $(STAGE_DIR)/linux-$(1) $(APP)
	@echo "  → $(DIST_DIR)/$(APP)-$(VERSION)-linux-$(1).tar.gz"
endef

dist-linux: dist-linux-amd64 dist-linux-arm64 ## Paquets Linux (x86-64 et ARM64)

dist-linux-amd64: ## Paquet Linux x86-64 (.tar.gz)
	$(call linux_package,amd64)

dist-linux-arm64: ## Paquet Linux ARM64 (.tar.gz)
	$(call linux_package,arm64)

# ------------------------------------------------------------------- tout

dist: dist-windows dist-linux ## Construit tous les paquets possibles sur cette machine
ifeq ($(HOST_OS),darwin)
dist: dist-macos
endif

checksums: ## Écrit dist/SHA256SUMS.txt
	@cd $(DIST_DIR) && \
		( command -v shasum >/dev/null && shasum -a 256 $(APP)-* > SHA256SUMS.txt \
		  || sha256sum $(APP)-* > SHA256SUMS.txt )
	@echo "  → $(DIST_DIR)/SHA256SUMS.txt"

# ------------------------------------------------- installation locale

install: ## Installe sur cette machine (macOS : ~/Applications, Linux : ~/.local)
ifeq ($(HOST_OS),darwin)
	$(MAKE) dist-macos-universal
	mkdir -p "$(MACOS_APPS)"
	rm -rf "$(MACOS_APPS)/$(APP).app"
	cp -R $(STAGE_DIR)/macos-universal/$(APP).app "$(MACOS_APPS)/$(APP).app"
	@echo "Installé : $(MACOS_APPS)/$(APP).app"
else
	$(MAKE) build
	mkdir -p "$(PREFIX)/lib/$(BIN)" "$(PREFIX)/bin" "$(PREFIX)/share/applications"
	install -m 0755 $(BUILD_DIR)/$(BIN) "$(PREFIX)/lib/$(BIN)/$(BIN)"
	rm -rf "$(PREFIX)/lib/$(BIN)/Resources"
	cp -R Resources "$(PREFIX)/lib/$(BIN)/Resources"
	ln -sf "$(PREFIX)/lib/$(BIN)/$(BIN)" "$(PREFIX)/bin/$(BIN)"
	sed -e 's/@VERSION@/$(VERSION)/g' \
		-e 's|@EXEC@|$(PREFIX)/lib/$(BIN)/$(BIN)|' \
		-e 's|@ICON@|$(PREFIX)/lib/$(BIN)/Resources/Image_logiciel.png|' \
		packaging/hb_go_version5.desktop.in > "$(PREFIX)/share/applications/$(BIN).desktop"
	@echo "Installé : $(PREFIX)/bin/$(BIN)"
endif

uninstall: ## Désinstalle de cette machine
ifeq ($(HOST_OS),darwin)
	rm -rf "$(MACOS_APPS)/$(APP).app"
else
	rm -rf "$(PREFIX)/lib/$(BIN)" "$(PREFIX)/bin/$(BIN)" \
		"$(PREFIX)/share/applications/$(BIN).desktop"
endif
	@echo "Désinstallation terminée."
