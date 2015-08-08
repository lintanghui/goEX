.PHONY: .FORCE

PROGS = client \
	server \

SRCDIR = ./src

all: $(PROGS)

$(PROGS):
	go install $@

clean:
	rm -rf bin pkg

fmt:
	go fmt $(SRCDIR)/...
