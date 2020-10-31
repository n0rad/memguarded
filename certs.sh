#!/bin/bash
set -x
set -e

PASS="test"               # passphrase
EMAIL="test@random.com"   # mail
OU="R&D"                  # OrganizationalUnit
O="Company Ltd."          # Organization
SUBJECT="/OU=$OU/O=$O"

#CN="Sample Cert"          # CommonName
#L="Dublin 4"              # Locality
#S="Dublin"                # StateOrProvinceName
#C="IE"                    # CountryName
#SUBJECT="/C=$C/ST=$S/L=$L/O=$O/OU=$OU/CN=$CN/emailAddress=$EMAIL"


rm -rf certs
mkdir certs
cd certs

echo "make CA"
openssl req -new -x509 -days 365 -keyout ca.key -out ca.pem -subj "$SUBJECT" -passout pass:$PASS

echo "make server cert"
openssl req -new -nodes -x509 -out server.pem -keyout server.key -days 365 -subj "$SUBJECT"

echo "make client cert"
#openssl req -new -nodes -x509 -out client.pem -keyout client.key -days 3650 -subj "$SUBJECT"

#openssl genrsa -aes256 -out client.key 4096
openssl genpkey -algorithm RSA -aes-256-cbc -pkeyopt rsa_keygen_bits:4096 -out client.key
echo "00" > ca.srl
openssl req -sha512 -key client.key -new -out client.req -subj "$SUBJECT"
# Adding -addtrust clientAuth makes certificates Go can't read
openssl x509 -req -days 365 -in client.req -CA ca.pem -CAkey ca.key -passin pass:$PRIVKEY -out client.pem # -addtrust clientAuth

openssl x509 -extfile ../openssl.conf -extensions ssl_client -req -days 365 -in client.req -CA ca.pem -CAkey ca.key -passin pass:$PASS -out client.pem
