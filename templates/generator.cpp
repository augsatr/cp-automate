#include <bits/stdc++.h>
using namespace std;

/* Generate random test cases for stress testing.
   Usage: raga -stress -brute=Brute -optimized=Fast -gen=Generator */

int main() {
    ios_base::sync_with_stdio(0);
    cin.tie(0);

    mt19937 rng(chrono::steady_clock::now().time_since_epoch().count());

    int n = uniform_int_distribution<int>(1, 10)(rng);
    cout << n << "\n";
    for (int i = 0; i < n; i++) {
        cout << uniform_int_distribution<int>(1, 100)(rng) << " \n"[i == n - 1];
    }
}
