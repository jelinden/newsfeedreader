#!bin/bash
DATE_STRING=$(date +"%s")
echo "deleting old minified files"
rm public/css/*.min.css
rm public/js/*.min.js
echo "minifying new files"
cd public/css
yuicompressor uutispuro.css > uutispuro-$DATE_STRING.min.css
cd ../js
yuicompressor uutispuro.js > uutispuro-$DATE_STRING.min.js
cd ../..
echo "creating manifest file"
echo "{
  \"public/css/uutispuro.css\": \"public/css/uutispuro-$DATE_STRING.min.css\",
  \"public/js/uutispuro.js\": \"public/js/uutispuro-$DATE_STRING.min.js\"
}" > manifest.json
cat manifest.json
