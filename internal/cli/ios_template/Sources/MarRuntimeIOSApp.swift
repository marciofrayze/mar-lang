import SwiftUI

@main
struct MarRuntimeIOSApp: App {
    @StateObject private var model = AppViewModel()

    var body: some Scene {
        WindowGroup {
            RootView(model: model)
                .task {
                    await model.start()
                }
        }
    }
}
