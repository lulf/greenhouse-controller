FROM fedora-minimal:latest

RUN microdnf -y update && microdnf -y install qpid-proton-c && microdnf -y clean all
ADD build/greenhouse-controller /

ENTRYPOINT ["/greenhouse-controller"]
