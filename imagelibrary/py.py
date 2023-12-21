import json
import os
import shutil


with open("./dalle.json", "r") as f:
    data = json.load(f)

filelist=os.listdir("./2")
for fichier in filelist[:]: # filelist[:] makes a copy of filelist.
    if not(fichier.endswith(".png")):
        filelist.remove(fichier)

newData = {}
for  i, f in enumerate(data):
    newName = f[37:]
    print(newName)
    file = "./2/" + str(i% 8 + 1) + ".png"

    print(file)

    shutil.copy(file, newName)
