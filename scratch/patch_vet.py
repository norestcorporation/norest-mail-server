import re

def fix_file(filepath):
    with open(filepath, "r") as f:
        content = f.read()

    # Find "resp, err := client.Do(req)" and "defer resp.Body.Close()"
    # It might be using resp before checking err
    # I'll just use a simple regex for the common mistake:
    # resp, err := client.Do(req)
    # defer resp.Body.Close()
    # if err != nil
    
    content = re.sub(
        r"(resp, err := [^\n]+\n)\s*(defer resp\.Body\.Close\(\))\n\s*(if err != nil {\n\s*return[^\n]+\n\s*})",
        r"\1\3\n\t\2",
        content
    )
    
    with open(filepath, "w") as f:
        f.write(content)

fix_file("scripts/verify-chapter3/main.go")
fix_file("scripts/verify-chapter2/main.go")
