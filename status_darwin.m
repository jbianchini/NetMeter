#import <Cocoa/Cocoa.h>

extern void GoResetCounter(void);

static NSStatusItem *statusItem;
static NSMenuItem *detailsItem;

@interface AppDelegate : NSObject <NSApplicationDelegate>
@end

@implementation AppDelegate
- (void)applicationDidFinishLaunching:(NSNotification *)notification {
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];

    statusItem = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
    statusItem.button.title = @"↓ 0 MB ↑ 0 MB";

    NSMenu *menu = [[NSMenu alloc] init];

    detailsItem = [[NSMenuItem alloc] initWithTitle:@"Loading…" action:nil keyEquivalent:@""];
    [detailsItem setEnabled:NO];
    [menu addItem:detailsItem];

    [menu addItem:[NSMenuItem separatorItem]];

    NSMenuItem *resetItem = [[NSMenuItem alloc] initWithTitle:@"Reset counter" action:@selector(resetCounter:) keyEquivalent:@"r"];
    [resetItem setTarget:self];
    [menu addItem:resetItem];

    NSMenuItem *quitItem = [[NSMenuItem alloc] initWithTitle:@"Quit" action:@selector(quit:) keyEquivalent:@"q"];
    [quitItem setTarget:self];
    [menu addItem:quitItem];

    statusItem.menu = menu;
}

- (void)resetCounter:(id)sender {
    GoResetCounter();
}

- (void)quit:(id)sender {
    [NSApp terminate:nil];
}
@end

void setStatusTitle(const char *title) {
    NSString *s = [NSString stringWithUTF8String:title];
    dispatch_async(dispatch_get_main_queue(), ^{
        if (statusItem && statusItem.button) {
            statusItem.button.title = s;
        }
    });
}

void setMenuDetails(const char *details) {
    NSString *s = [NSString stringWithUTF8String:details];
    dispatch_async(dispatch_get_main_queue(), ^{
        if (detailsItem) {
            detailsItem.title = s;
        }
    });
}

void runStatusApp(void) {
    @autoreleasepool {
        NSApplication *app = [NSApplication sharedApplication];
        AppDelegate *delegate = [[AppDelegate alloc] init];
        [app setDelegate:delegate];
        [app run];
    }
}
