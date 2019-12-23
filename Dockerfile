FROM fedora-minimal:latest

ADD build/plant-controller /

ENTRYPOINT ["/plant-controller"]
