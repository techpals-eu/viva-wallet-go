Παρακάτω παραθέτω τα bullets και δίπλα στο κάθε ένα έχω συμπληρώσει τα credentials.

 

Για login στο viva e-banking

URL https://demo.vivapayments.com/en/signin
Username <tdb>
Password <tdb>
 

Σε συνεχεία του χθεσινού μας call, σας παραθέτω μια σύνοψη των όσων είπαμε:

1. Η Smart Checkout είναι μια web based σελίδα στην οποία ανακατευθύνεται ο χρήστης για να ολοκληρώσει την πληρωμή του.
2. Το URL της σελίδας πληρωμών της viva αποτελείται από τα παρακάτω μέρη
  a. Base URL (static parameter)
  b. Payment Order (dynamic parameter)

 

Παράδειγμα

https://demo.vivapayments.com/web2?ref=5522243373579129&color=d5791a

• Δημιουργία Payment Order
 

Για την δημιουργία του payment order (είναι το μόνο δυναμικό μέρος του checkout web URL) θα πρέπει - πριν το user redirection - να εκτελέσετε τo POST Api Call

• Authentication: Oauth2.0
• Credentials
    o Client ID: 2xjaujh2ars66jpx20x3zygqllbzg8e1rpjz0k53ryep7.apps.vivapayments.com
    o Client secret: <tdb>


• Endpoint : https://demo-api.vivapayments.com/checkout/v2/isv/orders
• Body
{
  "amount": 10,  // --> Το τελικό ποσό που θα χρεωθεί ο πελάτης
  "isvAmount": // --> H προμήθεια του ISV.
  "customerTrns": "Πληρωμή καφέ",  // -->  Τι θα εμφανίζεται ως περιγραφή στην smart checkout
  "customer": {
    "email": marina@viva.com, // --> το email του χρήστη
    "fullName": "Marina Athanasopoulou", // --> το full name του χρήστη
    "phone": "6991111111", // --> το κινητό του χρήστη
    "countryCode": "GR", // --> ορίζεται η χώρα του πελατη
    "requestLang": "el-GR" // — > ορίζεται η default γλώσσα στην οποία θα προβάλλεται η σελίδα πληρωμών
  },
  "currencyCode": 978,  // — > Euro currency code
  "paymentTimeout": 600, // --> Time out 10 λεπτών (ειναι σε secs)
  "disableCash": true, // — > παράμετρος που ορίζεται ότι δεν είναι τα μετρητά αποδεκτός τρόπος πληρωμής
}

• Ορισμός color

Ο κωδικός χρώματος είναι hex colour και μπορεί να ορίζετε ξεχωριστό value σε κάθε redirect.
• Λήψη αποτελέσματος συναλλαγής

Για την λήψη του αποτελέσματος της συναλλαγής, δύναται να χρησιμοποιηθούν δύο τρόποι
Πιο συγκεκριμένα:

1. https://developer.viva.com/isv-partner-program/payment-isv-api/#tag/Payments/paths/~1api~1orders~1{order_code}/get με ορισμό το payment order, threshold ανά 3 secs και max time 10 λεπτά.

Credentials

Reseller ID: <TBD>
Merchant ID: 393969b6-c18e-4770-ba9a-2838c2beafee (Είναι μεταβλητό, στην συγκεκριμένη περίπτωση είναι το Μerchant id του demo account σας)
Reseller API Key: <TDB>

Username = <tdb:tdb>
Password = <tbd>


B) EFT POS through Cloud Rest API Protocol

Όπως συζητήθηκε, η επικοινωνία μέσω του PMS σας με τα τερματικά της Viva θα γίνεται over to the internet, με χρήση HTTPS api calls.

{
  "sessionId": "viva321", // — > Ορίστε ένα unique Session ID (UUID)
  "terminalId": "16013000", // --> Ορίστε το TID στο οποίο θα αποστέλλετε το αίτημα χρέωσης. Συστήνουμε κάθε σταθμός εξυπηρέτησης να είναι setup με ένα και μόνο τερματικό.
  "cashRegisterId": "tameio1",  // --> Unique Identification Key για να διακρίνετε από ποιο σταθμό εξυπηρέτησης έγινε trigger η εκάστοτε Χ συναλλαγή
  "amount": 734, // — > Το τελικό ποσό που θα χρεωθεί ο πελάτης
  "currencyCode": "978", // — > Το νόμισμα χρέωσης , θα πρέπει να είναι πάντα euro
  "merchantReference": "bookingnumberabc",  // — > Unique Identification Key για το δέσιμο της κράτησης με την πληρωμή. Σημείωση: Με βάση αυτό το key θα κάνετε αναζήτηση στο viva ebanking σας και θα δένεται η εκάστοτε πληρωμή με την κράτηση. Συστήνουμε να ορίζετε το κωδικό κράτησης.
  "customerTrns": "paymentforcustomer", // — > Κάποια περιγραφή για τον πελάτη η οποία αντιτροπτίζει ξεκάθαρα το τι αφορά η χρέωση
  "tipAmount": 0, // --> Ορίστε το tip που επιθυμεί ο πελάτης να σας αποδώσει για την πληρωμή αυτή. Προσοχή: Το tip είναι ποσό excluded από το amount parameter. Δηλαδή, αν το ποσό είναι 100 ευρώ και το tip 10 ευρώ, εμείς ως viva κάνουμε το sum up από την πλευρά μας.
  "showTransactionResult": true, // --> Παράμετρος με την οποία ορίζετε το αποτέλεσμα της συναλλαγής να είναι διακριτό στο τερματικό μετά την διαχείριση αυτής
  "showReceipt": true  // --> Παράμετρος με την οποία ορίζετε η δυνατότητα εκτύπωσης της απόδειξης της συναλλαγής από το τερματικό.
  "isvDetails" {  // --> Παράμετροι σχετικά με τον ISV
       "amount" // — > Η χρέωση ISV που εφαρμόζεται στη συναλλαγή από τον συνεργάτη ISV (δεν προστίθεται στο συνολικό ποσό πώληση)
       "terminalMerchantId" // -- > Το  id  του εμπόρου που είναι ενεργοποιημένο στo POS - εφαρμογή/συσκευή
  }
}

 

• Για την επιστροφή ποσού σε οποιαδήποτε άλλη κάρτα,  θα πρέπει να κάνετε trigger το παρακάτω Api Call

https://developer.viva.com/apis-for-point-of-sale/card-terminals-devices/rest-api/eft-pos-api-documentation/#tag/Make-Transactions/paths/~1ecr~1isv~1v1~1transactions:refund/post

• Για την λήψη του αποτελέσματος της συναλλαγής (είτε αιτήματος χρέωσης - είτε αιτήματος ακύρωσης), θα πρέπει να κάνετε trigger το Api Call που θα διακρίνετε στο παρακάτω σύνδεσμο.

https://developer.viva.com/apis-for-point-of-sale/card-terminals-devices/rest-api/eft-pos-api-documentation/#tag/Session-Details/paths/~1ecr~1isv~1v1~1sessions~1{sessionId}/get

Θα πρέπει μετά την αποστολή του αιτήματος χρέωσης, να ξεκινήσετε ένα pooling ανά 3 δευτερόλεπτα με max time out τα 3 λεπτά, καθώς δεν γνωρίζετε πότε/σε πόσο χρόνο ο πελάτης θα ολοκληρώσει την συναλλαγή του.

Προτεινόμενα Webhooks

Τεκμηρίωση: https://developer.viva.com/isv-partner-program/payment-isv-api/#tag/Webhooks

Ο server της Viva.com θα αποστέλλει JSON payload στο webhook endpoint που θα μας υποδείξετε.

Transaction Payment Created (EventTypeId: 1796)  Ενεργοποιείται όταν δημιουργηθεί επιτυχώς μία πληρωμή.


https://developer.viva.com/webhooks-for-payments/transaction-payment-created/


Transaction Failed (EventTypeId: 1798)
Ενεργοποιείται όταν μία πληρωμή αποτύχει.

https://developer.viva.com/webhooks-for-payments/transaction-failed/

Account Connected (EventTypeId: 8193)

Ενεργοποιείται όταν συνδεθεί επιτυχώς ένας λογαριασμός.

https://developer.viva.com/webhooks-for-payments/account-connected/

Account Verification Status Changed (EventTypeId: 8194)  Ενεργοποιείται όταν αλλάξει η κατάσταση επαλήθευσης (KYC/KYB) ενός λογαριασμού.

https://developer.viva.com/webhooks-for-payments/account-verification-status-changed/

Transaction Reversal Created (EventTypeId 1797)

[Transaction Reversal Created | Viva.com Developer Portal](https://developer.viva.com/webhooks-for-payments/transaction-reversal-created/)
