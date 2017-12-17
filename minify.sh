#!bin/bash
DATE_STRING=$(date +"%s")
echo "deleting old minified files"
rm public/css/*.min.css
rm public/js/*.min.js
echo "minifying new files"
cd public/css
purifycss pure-0.6.0.css uutispuro.css ../js/uutispuro.js ../html/*.html ../html/*.tmpl --min --info --out uutispuro-purified.min.css
yuicompressor uutispuro-purified.min.css > uutispuro-$DATE_STRING.min.css
#yuicompressor uutispuro.css >> uutispuro-$DATE_STRING.min.css
cd ../js
yuicompressor uutispuro.js > uutispuro-$DATE_STRING.min.js
cd ../..
echo "creating manifest file"
echo "{
  \"public/css/uutispuro.css\": \"public/css/uutispuro-$DATE_STRING.min.css\",
  \"public/js/uutispuro.js\": \"public/js/uutispuro-$DATE_STRING.min.js\"
}" > manifest.json
cat manifest.json
