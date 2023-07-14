def escape():
    s = """Roblox	com.roblox.client"""
    return s.replace("\n", "\\n").replace("\t", "\\t")

if __name__ == "__main__":
    print(escape())