echo "UPLOADING $1 channel $2 $3"
mkdir -p "/backup/$1"
cp $2 "/backup/$1"
cp $3 "/backup/$2"