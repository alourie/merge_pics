#!/bin/python
import os
import sys
import shutil
import datetime
import exifread
import hashlib


def fhash(filepath):
    BLOCKSIZE = 65536
    hasher = hashlib.sha256()
    with open(filepath, 'rb') as afile:
        buf = afile.read(BLOCKSIZE)
        while len(buf) > 0:
            hasher.update(buf)
            buf = afile.read(BLOCKSIZE)

    return hasher.hexdigest()


if __name__ == "__main__":
    names = []
    basedir = os.path.expanduser("~/Pictures/Australia")
    for root, dirs, files in os.walk("."):
        names.extend(filen for filen in files if filen.endswith(".jpg"))

    for name in names:
        with open(name, 'rb') as f:
            tags = exifread.process_file(f)
            if 'EXIF DateTimeOriginal' in tags:
                a = tags['EXIF DateTimeOriginal']
            elif 'Image DateTime' in tags:
                a = tags['Image DateTime']
            else:
                print "File " + name + " has no exif"
                sys.exit(1)

            date = datetime.datetime.strptime(a.values, '%Y:%m:%d %H:%M:%S')
            new_dir = os.path.join(basedir, str(date.year), str(date.month) if date.month > 9 else "0" + str(date.month), str(date.day))
            new_name = os.path.join(new_dir, name)
            picname, _ = name.split(".")
            # print "New name: " + new_name

            if os.path.exists(new_name):
                # if yes, compare the hashes.
                with open(name, 'r') as orig, open(new_name, 'r') as newn:
                    if fhash(orig) == fhash(newn):
                        os.remove(name)
                    else:
                        new_name = os.path.join(new_dir, picname + "_01.jpg" + picname)
                        shutil.copy(name, new_name)
            else:
                # if not, move the file
                if not os.path.exists(new_dir):
                    os.makedirs(new_dir)
                shutil.move(name, new_name)
