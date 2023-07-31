def escape():
    s = """TubiTV	com.tubitv"""
    return s.replace("\n", "\\n").replace("\t", "\\t")

if __name__ == "__main__":
    print(escape())