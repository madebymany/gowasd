install:
	cd "$(SOURCE_ROOT)/../gopath/bin" && \
	  install -C -S -m=0755 -o root -g root wasd /usr/bin

.PHONY: install
