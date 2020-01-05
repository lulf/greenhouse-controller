FROM fedora-minimal:latest

ADD build/greenhouse-controller /

ENTRYPOINT ["/greenhouse-controller"]
