Here's a **`README.md`** for your project, documenting how to set up, run, and understand the **watch-tower** Kubernetes controller. 🚀

---

### 📄 **README.md**
```markdown
# 🏰 Watch-Tower

## 📌 Overview
**Watch-Tower** is a **Kubernetes controller** written in **Go** that:
- **Monitors an AutomationController Custom Resource (CRD)**.
- **Checks the PostgreSQL database role (Primary/Standby)**.
- **Automatically scales the AutomationController** based on:
  - If the DB is **Primary**, it sets `spec.replicas = watch-tower/replicas` annotation.
  - If the DB is **Standby**, it sets `spec.replicas = 0`.

## 🚀 Features
✅ **Continuously monitors AutomationController CRD**  
✅ **Detects PostgreSQL role (Primary/Standby)**  
✅ **Reads `watch-tower/replicas` annotation for scaling**  
✅ **Uses Kubernetes API to patch `spec.replicas`**  
✅ **Runs in an infinite loop for real-time updates**  

---

## ⚙️ **Setup & Installation**

### **1️⃣ Prerequisites**
Ensure you have the following installed:
- **Go 1.20+** (`go version`)
- **Kubernetes CLI (kubectl)** (`kubectl version --client`)
- **OpenShift CLI (oc)** _(if using OpenShift)_ (`oc version`)
- **PostgreSQL running in Kubernetes**

### **2️⃣ Clone the Repository**
```bash
git clone https://github.com/your-org/watch-tower.git
cd watch-tower
```

### **3️⃣ Install Dependencies**
```bash
go mod tidy
```

---

## ⚡ **Running the Controller**
### **1️⃣ Set Environment Variables**
```bash
export DB_CREDENTIAL_PATH="/srv/db_credential"
export AAP_NAMESPACE="aap"
export KUBECONFIG=~/.kube/config
```

### **2️⃣ Run the Controller**
```bash
go run main.go
```

---

## 🔄 **How It Works**
1. **Loops every 30 seconds** to check:
   - **PostgreSQL role** (`Primary` or `Standby`).
   - **AutomationController resources in the namespace**.

2. **Reads the `watch-tower/replicas` annotation**:
   - If **missing**, it skips the resource.
   - If **invalid**, it logs a warning.

3. **Decides the correct replica count**:
   - If **Primary**, it uses the annotation value.
   - If **Standby**, it scales **down to `0`**.

4. **Checks the current `spec.replicas`**:
   - ✅ If **already correct**, it **skips patching**.
   - 🔄 If **different**, it **patches the CRD**.

---

## 📄 **Example YAML (AutomationController)**
Here’s an example **AutomationController CRD** with the required annotation:
```yaml
apiVersion: automationcontroller.ansible.com/v1beta1
kind: AutomationController
metadata:
  name: example
  namespace: aap
  annotations:
    watch-tower/replicas: "3"
spec:
  replicas: 3
```
- The annotation `watch-tower/replicas: "3"` tells Watch-Tower how many replicas to use **if the DB is Primary**.

---

## 📌 **Expected Output**
### **If the database is Primary**
```
✅ Using local Kubeconfig
🚀 Starting Watch Loop for AutomationController...

🔄 Checking AutomationController and Database Role...
✅ Successfully connected to PostgreSQL!
🔍 Database Role: Primary
✅ No update needed for example (replicas already set to 3)
```

### **If the database is Standby**
```
✅ Using local Kubeconfig
🚀 Starting Watch Loop for AutomationController...

🔄 Checking AutomationController and Database Role...
✅ Successfully connected to PostgreSQL!
🔍 Database Role: Standby
✅ Successfully patched example: spec.replicas = 0
```

---

