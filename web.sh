git clone https://github.com/maxischmaxi/league-game-web web
cd web
npm install
npm run build

echo "Copying files to public folder"
cp -r ./dist ../public

cd ..
rm -rf web
echo "Done"
