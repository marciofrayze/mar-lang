import Foundation

final class SessionStore {
    private let key = "MarRuntimeIOS.Session"

    func load() -> SessionSnapshot? {
        guard
            let data = UserDefaults.standard.data(forKey: key),
            let session = try? JSONDecoder().decode(SessionSnapshot.self, from: data)
        else {
            return nil
        }
        return session
    }

    func save(_ session: SessionSnapshot) {
        if let data = try? JSONEncoder().encode(session) {
            UserDefaults.standard.set(data, forKey: key)
        }
    }

    func clear() {
        UserDefaults.standard.removeObject(forKey: key)
    }
}
