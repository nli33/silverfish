# used in early development to compare 2 perft results (typically stockfish vs our engine)

def compare_files(file1_path, file2_path):
    with open(file1_path, 'r') as f1:
        lines1 = set(line.strip() for line in f1 if line.strip())

    with open(file2_path, 'r') as f2:
        lines2 = set(line.strip() for line in f2 if line.strip())

    only_in_file1 = lines1 - lines2
    only_in_file2 = lines2 - lines1

    if not only_in_file1 and not only_in_file2:
        print("The files have the same content (order ignored).")
    else:
        if only_in_file1:
            print("Lines only in file 1:")
            for line in sorted(only_in_file1):
                print("  ", line)
        if only_in_file2:
            print("Lines only in file 2:")
            for line in sorted(only_in_file2):
                print("  ", line)

# put perft result in perft1 and perft2
compare_files('perft1', 'perft2')

