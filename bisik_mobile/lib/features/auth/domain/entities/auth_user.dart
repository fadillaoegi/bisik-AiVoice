/// Peran yang memakai aplikasi. Nasabah sengaja tidak ada di sini:
/// dia tidak pernah memegang perangkat.
enum Role { officer, supervisor }

Role roleFromString(String? raw) =>
    raw == 'supervisor' ? Role.supervisor : Role.officer;

class AuthUser {
  const AuthUser({required this.id, required this.name, required this.role});

  final String id;
  final String name;
  final Role role;
}
