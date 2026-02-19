#!/bin/bash

echo "Checking if MongoDB container is running..."
if [ ! "$(docker ps -q -f name=mongodb)" ]; then
    if [ "$(docker ps -aq -f name=mongodb)" ]; then
        echo "Starting existing mongodb container..."
        docker start mongodb
    else
        echo "MongoDB container not found. Please ensure 'docker run' command finished."
        exit 1
    fi
fi

echo "Waiting for MongoDB to accept connections..."
# Loop until we can connect
# We try 'mongosh' first, then 'mongo'
until docker exec mongodb mongosh --eval "db.runCommand('ping').ok" &> /dev/null || docker exec mongodb mongo --eval "db.runCommand('ping').ok" &> /dev/null
do
    echo "MongoDB is not ready yet... sleeping 2s"
    sleep 2
done

echo "MongoDB is up! Seeding admin user..."

# Seed the user
docker exec mongodb mongosh test --eval '
try {
  db.users.drop(); 
} catch(e) {}
db.users.insertOne({username: "admin", password: "12345"});
print("User created: admin / 12345");
'

echo "========================================="
echo "Database setup complete!"
echo "You can now run: go run main.go"
echo "========================================="
