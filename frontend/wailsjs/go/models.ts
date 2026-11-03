export namespace app {
	
	export class Info {
	    running: boolean;
	    port: number;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.port = source["port"];
	        this.error = source["error"];
	    }
	}

}

